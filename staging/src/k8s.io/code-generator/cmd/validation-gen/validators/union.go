/*
Copyright 2021 The Kubernetes Authors.
Modifications copyright 2025 <Your Name/Org for these changes>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package validators

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/gengo/v2/parser/tags"
	"k8s.io/gengo/v2/types"
)

// Define these constants at the package level
const (
	unionDiscriminatorTagName = "k8s:unionDiscriminator"
	unionMemberTagName        = "k8s:unionMember"
)

var unionTagValidScopes = sets.New(ScopeField)

var discriminatedUnionValidator = types.Name{Package: libValidationPkg, Name: "DiscriminatedUnion"}
var unionValidator = types.Name{Package: libValidationPkg, Name: "Union"}

var newDiscriminatedUnionMembership = types.Name{Package: libValidationPkg, Name: "NewDiscriminatedUnionMembership"}
var newUnionMembership = types.Name{Package: libValidationPkg, Name: "NewUnionMembership"}

// libValidationPkg must be defined, e.g., in another file in this package or via build tags.
// var libValidationPkg string = "k8s.io/apimachinery/pkg/api/validate" // Example

func init() {
	shared := map[*types.Type]unions{}
	RegisterTypeValidator(unionTypeValidator{shared})
	RegisterTagValidator(unionDiscriminatorTagValidator{shared})
	RegisterTagValidator(unionMemberTagValidator{shared})
}

type unionTypeValidator struct {
	shared map[*types.Type]unions
}

func (unionTypeValidator) Init(_ Config) {}

func (unionTypeValidator) Name() string {
	return "unionTypeValidator"
}

func isVirtualListMember(memberAny any) (*types.Member, bool) {
	var m *types.Member
	if typedM, ok := memberAny.(*types.Member); ok {
		m = typedM
	} else if valM, ok := memberAny.(types.Member); ok {
		m = &valM
	} else {
		return nil, false
	}
	return m, strings.HasPrefix(m.Name, virtualMemberPrefix)
}

func parseVirtualMemberMetadata(virtualMemberName string, elemType *types.Type) (string, map[string]string, error) {
	if !strings.HasPrefix(virtualMemberName, virtualMemberPrefix) {
		return "", nil, fmt.Errorf("not a virtual member name: %s", virtualMemberName)
	}
	trimmedName := strings.TrimPrefix(virtualMemberName, virtualMemberPrefix)
	parts := strings.SplitN(trimmedName, "_", 2)
	if len(parts) < 2 {
		return "", nil, fmt.Errorf("invalid virtual member name format (missing list field or selector): %s", virtualMemberName)
	}
	listFieldName := parts[0]
	selectorStr := parts[1]

	selectorParts := strings.Split(selectorStr, "_and_")
	selectorMap := make(map[string]string)
	for _, sp := range selectorParts {
		kv := strings.SplitN(sp, "_is_", 2)
		if len(kv) != 2 {
			return "", nil, fmt.Errorf("invalid selector part in virtual member name '%s': part '%s'", virtualMemberName, sp)
		}
		jsonKey := kv[0]
		value := kv[1]
		if getMemberByJSON(elemType, jsonKey) == nil {
			return "", nil, fmt.Errorf("virtual member selector key '%s' not found as JSON field in element type '%s'", jsonKey, elemType.Name)
		}
		selectorMap[jsonKey] = value
	}
	if len(selectorMap) == 0 {
		return "", nil, fmt.Errorf("no selector criteria parsed from virtual member name: %s", virtualMemberName)
	}
	return listFieldName, selectorMap, nil
}

func getGoFieldNameFromJSONKey(structType *types.Type, jsonKeyToFind string) (string, error) {
	s := nonPointer(nativeType(structType))
	if s.Kind != types.Struct {
		return "", fmt.Errorf("cannot get Go field name for JSON key '%s': type %s is not a struct", jsonKeyToFind, structType.Name)
	}

	for i := range s.Members {
		memberPtr := &s.Members[i]
		jsonAnnotation, ok := tags.LookupJSON(*memberPtr)

		effectiveJSONName := memberPtr.Name
		if ok && jsonAnnotation.Name != "" {
			effectiveJSONName = jsonAnnotation.Name
		}

		if effectiveJSONName == jsonKeyToFind {
			return memberPtr.Name, nil
		}
	}
	return "", fmt.Errorf("no field with effective JSON name %q in type %s", jsonKeyToFind, s.Name.String())
}

func (utv unionTypeValidator) GetValidations(context Context) (Validations, error) {
	result := Validations{}
	structType := nonPointer(nativeType(context.Type))

	if structType.Kind != types.Struct {
		return result, nil
	}

	unionsData, ok := utv.shared[structType]
	if !ok || len(unionsData) == 0 {
		return result, nil
	}

	unionNames := make([]string, 0, len(unionsData))
	for k := range unionsData {
		unionNames = append(unionNames, k)
	}
	slices.Sort(unionNames)

	for _, unionName := range unionNames {
		u := unionsData[unionName]
		if len(u.fieldMembers) == 0 && u.discriminator == nil {
			continue
		}

		var unionValidationFnName types.Name
		var initialArgs []any
		supportVarName := PrivateVar{Name: "UnionMembershipFor" + structType.Name.Name + unionName, Package: "local"}
		membershipArgs := u.fields

		if u.discriminator != nil {
			unionValidationFnName = discriminatedUnionValidator
			fullMembershipArgs := append([]any{*u.discriminator}, membershipArgs...)
			supportVar := Variable(supportVarName,
				Function(unionMemberTagName, DefaultFlags, newDiscriminatedUnionMembership, fullMembershipArgs...))
			result.Variables = append(result.Variables, supportVar)

			initialArgs = append(initialArgs, supportVarName)
			var discMemberType *types.Type
			var discMemberName string
			if dm, ok_dm := u.discriminatorMember.(*types.Member); ok_dm {
				discMemberType = dm.Type
				discMemberName = dm.Name
			} else if dmVal, ok_dm_val := u.discriminatorMember.(types.Member); ok_dm_val {
				discMemberType = dmVal.Type
				discMemberName = dmVal.Name
			} else if u.discriminatorMember != nil {
				return Validations{}, fmt.Errorf("discriminator member for union %s on type %s has unexpected type %T", unionName, structType.Name.String(), u.discriminatorMember)
			} else {
				return Validations{}, fmt.Errorf("discriminator name set but member is nil for union %s on type %s", unionName, structType.Name.String())
			}

			discriminatorAccessorLiteral := FunctionLiteral{
				Parameters: []ParamResult{},
				Results:    []ParamResult{{"", discMemberType}},
				Body:       fmt.Sprintf("return obj.%s", discMemberName),
			}
			initialArgs = append(initialArgs, discriminatorAccessorLiteral)
		} else {
			unionValidationFnName = unionValidator
			supportVar := Variable(supportVarName,
				Function(unionMemberTagName, DefaultFlags, newUnionMembership, membershipArgs...))
			result.Variables = append(result.Variables, supportVar)
			initialArgs = append(initialArgs, supportVarName)
		}

		var memberValueArgs []any
		for _, memberAny := range u.fieldMembers {
			var currentMember *types.Member
			if m, ok_m := memberAny.(*types.Member); ok_m {
				currentMember = m
			} else if mVal, okVal := memberAny.(types.Member); okVal {
				currentMember = &mVal
			} else {
				return Validations{}, fmt.Errorf("unexpected type %T in u.fieldMembers for union '%s'. Expected *types.Member or types.Member", memberAny, unionName)
			}

			if virtualMember, isVirtual := isVirtualListMember(currentMember); isVirtual {
				elemType := virtualMember.Type
				if elemType == nil {
					return Validations{}, fmt.Errorf("virtual member '%s' for union '%s' has nil Type", virtualMember.Name, unionName)
				}

				listFieldName, selectorMap, err := parseVirtualMemberMetadata(virtualMember.Name, elemType)
				if err != nil {
					return Validations{}, fmt.Errorf("error parsing metadata for virtual member '%s' (union '%s'): %w", virtualMember.Name, unionName, err)
				}

				listFieldInParent := getMemberPtr(structType, listFieldName)
				if listFieldInParent == nil {
					return Validations{}, fmt.Errorf("list field '%s' (from virtual member '%s') not found in struct '%s'", listFieldName, virtualMember.Name, structType.Name)
				}

				var functionLiteralBody strings.Builder
				functionLiteralBody.WriteString(fmt.Sprintf("if obj.%s == nil { return nil }\n", listFieldName))
				functionLiteralBody.WriteString(fmt.Sprintf("for _, listItem := range obj.%s {\n", listFieldName))
				var conditions []string
				for jsonKeyFromSelector, expectedValue := range selectorMap {
					goFieldName, err_gf := getGoFieldNameFromJSONKey(elemType, jsonKeyFromSelector)
					if err_gf != nil {
						return Validations{}, fmt.Errorf("error generating union validation for '%s' (virtual member '%s', selector key '%s'): %w", unionName, virtualMember.Name, jsonKeyFromSelector, err_gf)
					}
					conditions = append(conditions, fmt.Sprintf("listItem.%s == %q", goFieldName, expectedValue))
				}
				functionLiteralBody.WriteString(fmt.Sprintf("  if %s {\n", strings.Join(conditions, " && ")))
				functionLiteralBody.WriteString(fmt.Sprintf("    return &%s{}\n", elemType.Name.Name))
				functionLiteralBody.WriteString(fmt.Sprintf("  }\n"))
				functionLiteralBody.WriteString(fmt.Sprintf("}\n"))
				functionLiteralBody.WriteString(fmt.Sprintf("return nil"))

				argFuncLiteral := FunctionLiteral{
					Parameters: []ParamResult{},
					Results:    []ParamResult{{"", types.PointerTo(elemType)}},
					Body:       functionLiteralBody.String(),
				}
				memberValueArgs = append(memberValueArgs, argFuncLiteral)
			} else {
				functionLiteralBody := fmt.Sprintf("return obj.%s", currentMember.Name)
				realFieldAccessorLiteral := FunctionLiteral{
					Parameters: []ParamResult{},
					Results:    []ParamResult{{"", currentMember.Type}},
					Body:       functionLiteralBody,
				}
				memberValueArgs = append(memberValueArgs, realFieldAccessorLiteral)
			}
		}
		allFnArgs := append(initialArgs, memberValueArgs...)
		fn := Function(unionMemberTagName, DefaultFlags, unionValidationFnName, allFnArgs...)
		result.Functions = append(result.Functions, fn)
	}
	return result, nil
}

type unionDiscriminatorTagValidator struct {
	shared map[*types.Type]unions
}

func (unionDiscriminatorTagValidator) Init(_ Config)   {}
func (unionDiscriminatorTagValidator) TagName() string { return unionDiscriminatorTagName }

func (unionDiscriminatorTagValidator) ValidScopes() sets.Set[Scope] { return unionTagValidScopes }

func (udtv unionDiscriminatorTagValidator) GetValidations(context Context, _ []string, payload string) (Validations, error) {
	if context.Member == nil {
		return Validations{}, fmt.Errorf("%s: context.Member is nil, this tag must be on an actual struct field", udtv.TagName())
	}
	if t := nonPointer(nativeType(context.Type)); t != types.String {
		return Validations{}, fmt.Errorf("%s: can only be used on string types (field %s is %s)", udtv.TagName(), context.Member.Name, rootTypeString(context.Type, t))
	}

	p := &discriminatorParams{}
	if len(payload) > 0 {
		if err := json.Unmarshal([]byte(payload), &p); err != nil {
			return Validations{}, fmt.Errorf("error parsing JSON value for %s: %v (%q)", udtv.TagName(), err, payload)
		}
	}

	if context.Parent == nil {
		return Validations{}, fmt.Errorf("%s: context.Parent is nil for field %s, cannot define union", udtv.TagName(), context.Member.Name)
	}
	parentStructType := nonPointer(nativeType(context.Parent))
	if parentStructType.Kind != types.Struct {
		return Validations{}, fmt.Errorf("%s: context.Parent %s is not a struct type", udtv.TagName(), parentStructType.Name)
	}

	if udtv.shared[parentStructType] == nil {
		udtv.shared[parentStructType] = make(unions)
	}
	u := udtv.shared[parentStructType].getOrCreate(p.Union)

	var discriminatorFieldName string
	jsonAnnotation, ok := tags.LookupJSON(*context.Member)
	if ok && len(jsonAnnotation.Name) > 0 {
		discriminatorFieldName = jsonAnnotation.Name
	} else {
		discriminatorFieldName = context.Member.Name
	}
	u.discriminator = &discriminatorFieldName
	u.discriminatorMember = *context.Member
	return Validations{}, nil
}

func (udtv unionDiscriminatorTagValidator) Docs() TagDoc {
	return TagDoc{
		Tag:         udtv.TagName(),
		Scopes:      udtv.ValidScopes().UnsortedList(),
		Description: "Indicates that this field is the discriminator for a union.",
		Payloads: []TagPayloadDoc{{
			Description: "<json-object>",
			Docs:        "",
			Schema: []TagPayloadSchema{{
				Key:   "union",
				Value: "<string>",
				Docs:  "the name of the union, if more than one exists",
			}},
		}},
	}
}

type unionMemberTagValidator struct {
	shared map[*types.Type]unions
}

func (unionMemberTagValidator) Init(_ Config)                {}
func (unionMemberTagValidator) TagName() string              { return unionMemberTagName }
func (unionMemberTagValidator) ValidScopes() sets.Set[Scope] { return unionTagValidScopes }

func (umtv unionMemberTagValidator) GetValidations(context Context, _ []string, payload string) (Validations, error) {
	if context.Member == nil {
		return Validations{}, fmt.Errorf("%s: context.Member is nil. This should not happen if called via subfield that constructs a virtual member, or directly on a field.", umtv.TagName())
	}

	var fieldName string
	jsonAnnotation, ok := tags.LookupJSON(*context.Member)
	if !ok || jsonAnnotation.Name == "" {
		fieldName = context.Member.Name
	} else {
		fieldName = jsonAnnotation.Name
	}

	p := &memberParams{MemberName: context.Member.Name}
	if len(payload) > 0 {
		if err := json.Unmarshal([]byte(payload), &p); err != nil {
			return Validations{}, fmt.Errorf("error parsing JSON value for %s: %v (%q)", umtv.TagName(), err, payload)
		}
	}

	if context.Parent == nil {
		return Validations{}, fmt.Errorf("%s: context.Parent is nil for member %s, cannot define union", umtv.TagName(), context.Member.Name)
	}
	parentStructType := nonPointer(nativeType(context.Parent))
	if parentStructType.Kind != types.Struct {
		return Validations{}, fmt.Errorf("%s: context.Parent %s is not a struct type for member %s", umtv.TagName(), parentStructType.Name, context.Member.Name)
	}

	if umtv.shared[parentStructType] == nil {
		umtv.shared[parentStructType] = make(unions)
	}
	u := umtv.shared[parentStructType].getOrCreate(p.Union)

	u.fields = append(u.fields, [2]string{fieldName, p.MemberName})
	u.fieldMembers = append(u.fieldMembers, *context.Member)

	return Validations{}, nil
}

func (umtv unionMemberTagValidator) Docs() TagDoc {
	return TagDoc{
		Tag:         umtv.TagName(),
		Scopes:      umtv.ValidScopes().UnsortedList(),
		Description: "Indicates that this field is a member of a union.",
		Payloads: []TagPayloadDoc{{
			Description: "<json-object>",
			Docs:        "",
			Schema: []TagPayloadSchema{{
				Key:   "union",
				Value: "<string>",
				Docs:  "the name of the union, if more than one exists",
			}, {
				Key:     "memberName",
				Value:   "<string>",
				Docs:    "the discriminator value for this member",
				Default: "the field's name",
			}},
		}},
	}
}

type discriminatorParams struct {
	Union string `json:"union,omitempty"`
}

type memberParams struct {
	Union      string `json:"union,omitempty"`
	MemberName string `json:"memberName,omitempty"`
}

type union struct {
	fields              []any
	fieldMembers        []any
	discriminator       *string
	discriminatorMember any
}

type unions map[string]*union

func (us unions) getOrCreate(name string) *union {
	u, ok := us[name]
	if !ok {
		u = &union{}
		us[name] = u
	}
	return u
}

func getMemberPtr(structType *types.Type, goFieldName string) *types.Member {
	s := nonPointer(nativeType(structType))
	if s.Kind != types.Struct {
		return nil
	}
	for i := range s.Members {
		if s.Members[i].Name == goFieldName {
			return &s.Members[i]
		}
	}
	return nil
}

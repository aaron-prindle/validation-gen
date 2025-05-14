/*
Copyright 2025 The Kubernetes Authors.

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
	"sort"
	"strings"

	"k8s.io/apimachinery/pkg/api/validate/util"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/gengo/v2/parser/tags"
	"k8s.io/gengo/v2/types"
)

const (
	subfieldTagName = "k8s:subfield"
)

func init() {
	RegisterTagValidator(&subfieldTagValidator{})
}

type subfieldTagValidator struct {
	validator Validator
}

func (stv *subfieldTagValidator) Init(cfg Config) {
	stv.validator = cfg.Validator
}

func (subfieldTagValidator) TagName() string {
	return subfieldTagName
}

var subfieldTagValidScopes = sets.New(ScopeAny)

func (subfieldTagValidator) ValidScopes() sets.Set[Scope] {
	return subfieldTagValidScopes
}

type subfieldArgumentPayload struct {
	ListElems map[string]string `json:"listElems"`
	Flags     *struct {
		ShortCircuit *bool `json:"ShortCircuit,omitempty"`
		NonError     *bool `json:"NonError,omitempty"`
	} `json:"flags,omitempty"`
}

type parsedSubfieldArg struct {
	ListElems    map[string]string
	ShortCircuit bool
	NonError     bool
	SubName      string
}

var (
	validateSubfield            = types.Name{Package: libValidationPkg, Name: "Subfield"}
	validateListMapElementByKey = types.Name{Package: libValidationPkg, Name: "ListMapElementByKey"}
)

func (stv *subfieldTagValidator) GetValidations(context Context, args []string, payload string) (Validations, error) {
	if len(args) != 1 {
		return Validations{}, fmt.Errorf("requires exactly one arg")
	}
	configStr := args[0]

	parsedArg, err := parseSubfieldArg(configStr)
	if err != nil {
		return Validations{}, err
	}

	// This tag can apply to value and pointer fields, as well as typedefs
	// (which should never be pointers). We need to check the concrete type.
	t := nonPointer(nativeType(context.Type))
	fakeComments := []string{payload}

	if parsedArg.SubName != "" { // +k8s:subfield(subname) usage
		if t.Kind != types.Struct {
			return Validations{}, fmt.Errorf("can only be used on struct types")
		}

		subname := configStr
		submemb := getMemberByJSON(t, subname)
		if submemb == nil {
			return Validations{}, fmt.Errorf("no field for json name %q", subname)
		}

		result := Validations{}

		subContext := Context{
			Scope:  ScopeField,
			Type:   submemb.Type,
			Parent: t,
			Path:   context.Path.Child(subname),
			Member: submemb,
		}
		if validations, err := stv.validator.ExtractValidations(subContext, fakeComments); err != nil {
			return Validations{}, err
		} else {
			if len(validations.Variables) > 0 {
				return Validations{}, fmt.Errorf("variable generation is not supported")
			}

			for _, vfn := range validations.Functions {
				nilableStructType := context.Type
				if !isNilableType(nilableStructType) {
					nilableStructType = types.PointerTo(nilableStructType)
				}
				nilableFieldType := submemb.Type
				fieldExprPrefix := ""
				if !isNilableType(nilableFieldType) {
					nilableFieldType = types.PointerTo(nilableFieldType)
					fieldExprPrefix = "&"
				}

				getFn := FunctionLiteral{
					Parameters: []ParamResult{{"o", nilableStructType}},
					Results:    []ParamResult{{"", nilableFieldType}},
				}
				getFn.Body = fmt.Sprintf("return %so.%s", fieldExprPrefix, submemb.Name)
				f := Function(subfieldTagName, vfn.Flags, validateSubfield, subname, getFn, WrapperFunction{vfn, submemb.Type})
				result.Functions = append(result.Functions, f)
				result.Variables = append(result.Variables, validations.Variables...)
			}
		}
		return result, nil

	} else { // // +k8s:subfield({"listMapElems":{"..."}}) usage
		if t.Kind != types.Slice && t.Kind != types.Array {
			return Validations{}, fmt.Errorf("list access via 'listElems' can only be used on slice/array types, but tag is on field %s of type %s", context.Path.String(), t.Name.String())
		}

		elemT := nonPointer(nativeType(t.Elem))
		if elemT.Kind != types.Struct {
			return Validations{}, fmt.Errorf("elements of slice/array must be structs for key-based selection, but elements of field %s are %s", context.Path.String(), elemT.Name.String())
		}

		for k := range parsedArg.ListElems {
			if getMemberByJSON(elemT, k) == nil {
				return Validations{}, fmt.Errorf("element type %s (of list %s) has no field with JSON name %q", elemT.Name.String(), context.Path.String(), k)
			}
		}

		// TODO(aaron-prindle) currently can only select on slice fields that are strings, should have error related to this.

		// TODO(aaron-prindle) check to see if requirement to enforce context.Parent and/or context.Member are non-nil for path generation.

		// generates context path like Struct.Conditions[status="true",type="Approved"]
		subContextPath := util.GeneratePathForMap(parsedArg.ListElems)
		subContext := Context{
			Scope: ScopeField,
			Type:  elemT,
			// TODO(aaron-prindle) for +k8s:unionMember support need to plumb this.
			Parent: nil,
			Path:   context.Path.Key(subContextPath),
			// TODO(aaron-prindle) for +k8s:unionMember support need to plumb this.
			Member: nil,
		}

		if validations, err := stv.validator.ExtractValidations(subContext, fakeComments); err != nil {
			return Validations{}, err
		} else {

			result := Validations{}
			result.Variables = append(result.Variables, validations.Variables...)

			// MODIFIED: Create the GetSelectorMapForPathFunc literal
			selectorMapLiteralParts := []string{}

			sortedSelectorKeys := make([]string, 0, len(parsedArg.ListElems))
			for k := range parsedArg.ListElems {
				sortedSelectorKeys = append(sortedSelectorKeys, k)
			}
			sort.Strings(sortedSelectorKeys)
			for _, k := range sortedSelectorKeys {
				selectorMapLiteralParts = append(selectorMapLiteralParts, fmt.Sprintf("%q: %q", k, parsedArg.ListElems[k]))
			}
			getSelectorMapFnLiteral := FunctionLiteral{
				Parameters: []ParamResult{},
				Results:    []ParamResult{{"", &types.Type{Kind: types.Map, Key: types.String, Elem: types.String}}},
				Body:       fmt.Sprintf("return map[string]string{%s}", strings.Join(selectorMapLiteralParts, ", ")),
			}

			var matchFuncBody strings.Builder
			matchFuncBody.WriteString("if item == nil { return false }\n")
			var conditions []string
			for jsonKey, expectedValue := range parsedArg.ListElems {
				goFieldName, errGFN := getGoFieldNameFromJSONKeyForSubfield(elemT, jsonKey)
				if errGFN != nil {
					return Validations{}, fmt.Errorf("error creating MatchFunc for %s: could not get Go field name for JSON key '%s' in type '%s': %w", context.Path.String(), jsonKey, elemT.Name.String(), errGFN)
				}
				fieldMember := getMember(elemT, goFieldName)
				isPtrToString := false
				if fieldMember != nil && fieldMember.Type.Kind == types.Pointer && fieldMember.Type.Elem != nil && fieldMember.Type.Elem.Kind == types.String.Kind {
					isPtrToString = true
				}

				if isPtrToString {
					conditions = append(conditions, fmt.Sprintf("(item.%s != nil && *item.%s == %q)", goFieldName, goFieldName, expectedValue))
				} else {
					conditions = append(conditions, fmt.Sprintf("item.%s == %q", goFieldName, expectedValue))
				}
			}
			matchFuncBody.WriteString(fmt.Sprintf("return %s", strings.Join(conditions, " && ")))

			matchFnLiteral := FunctionLiteral{
				Parameters: []ParamResult{{"item", types.PointerTo(elemT)}},
				Results:    []ParamResult{{"", types.Bool}},
				Body:       matchFuncBody.String(),
			}

			for _, vfn := range validations.Functions {
				listMapCallFlags := DefaultFlags
				if parsedArg.ShortCircuit {
					listMapCallFlags |= ShortCircuit
				}
				if parsedArg.NonError {
					listMapCallFlags |= NonError
				}

				if listMapCallFlags.IsSet(NonError) && !listMapCallFlags.IsSet(ShortCircuit) {
					// TODO(aaron-prindle) FIXME - NonError: true, ShortCircuit: false causes code generator panic, remove NonError in this case.
					listMapCallFlags &^= NonError
				}

				f := Function(
					subfieldTagName,
					listMapCallFlags,
					validateListMapElementByKey,
					getSelectorMapFnLiteral,
					matchFnLiteral,
					WrapperFunction{vfn, elemT},
				)
				result.Functions = append(result.Functions, f)
			}
			return result, nil
		}

	}
}

func parseSubfieldArg(argStr string) (*parsedSubfieldArg, error) {
	var sap subfieldArgumentPayload

	if err := json.Unmarshal([]byte(argStr), &sap); err == nil {
		if len(sap.ListElems) == 0 {
			return nil, fmt.Errorf("'listElems' map cannot be empty")
		}

		var shortCircuit, nonError bool
		if sap.Flags != nil {
			if sap.Flags.ShortCircuit != nil {
				shortCircuit = *sap.Flags.ShortCircuit
			}
			if sap.Flags.NonError != nil {
				nonError = *sap.Flags.NonError
			}
		}
		return &parsedSubfieldArg{
			ListElems:    sap.ListElems,
			ShortCircuit: shortCircuit,
			NonError:     nonError,
		}, nil
	}
	if argStr == "" {
		return nil, fmt.Errorf("arg cannot be an empty string")
	}
	return &parsedSubfieldArg{
		SubName: argStr,
	}, nil
}

// Added getMember helper, as it's used by MatchFunc generation
func getMember(structType *types.Type, goFieldName string) *types.Member {
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

func getGoFieldNameFromJSONKeyForSubfield(structType *types.Type, jsonKeyToFind string) (string, error) {
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

// TODO(aaron-prindle) update Docs() w/ info for new listMapElem support
func (stv subfieldTagValidator) Docs() TagDoc {
	doc := TagDoc{
		Tag:         stv.TagName(),
		Scopes:      stv.ValidScopes().UnsortedList(),
		Description: "Declares a validation for a subfield of a struct.",
		Args: []TagArgDoc{{
			Description: "<field-json-name>",
		}},
		Docs: "The named subfield must be a direct field of the struct, or of an embedded struct.",
		Payloads: []TagPayloadDoc{{
			Description: "<validation-tag>",
			Docs:        "The tag to evaluate for the subfield.",
		}},
	}
	return doc
}

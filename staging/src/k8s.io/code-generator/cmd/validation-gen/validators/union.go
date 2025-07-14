/*
Copyright 2021 The Kubernetes Authors.

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
	"regexp"
	"slices"
	"strings"

	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/code-generator/cmd/validation-gen/util"
	"k8s.io/gengo/v2/codetags"
	"k8s.io/gengo/v2/parser/tags"
	"k8s.io/gengo/v2/types"
)

var discriminatedUnionValidator = types.Name{Package: libValidationPkg, Name: "DiscriminatedUnion"}
var unionValidator = types.Name{Package: libValidationPkg, Name: "Union"}

var newDiscriminatedUnionMembership = types.Name{Package: libValidationPkg, Name: "NewDiscriminatedUnionMembership"}
var newUnionMembership = types.Name{Package: libValidationPkg, Name: "NewUnionMembership"}

func init() {
	// Unions are comprised of multiple tags, which need to share information
	// between them.  The tags are on struct fields, but the validation
	// actually pertains to the struct itself.
	shared := map[string]unions{}
	RegisterTypeValidator(unionTypeOrFieldValidator{shared})
	RegisterFieldValidator(unionTypeOrFieldValidator{shared})
	RegisterTagValidator(unionDiscriminatorTagValidator{shared})
	RegisterTagValidator(unionMemberTagValidator{shared})
}

type unionTypeOrFieldValidator struct {
	shared map[string]unions
}

func (unionTypeOrFieldValidator) Init(_ Config) {}

func (unionTypeOrFieldValidator) Name() string {
	return "unionTypeOrFieldValidator"
}

func BAR() {}
func (utfv unionTypeOrFieldValidator) GetValidations(context Context) (Validations, error) {
	result := Validations{}

	// Gengo does not treat struct definitions as aliases, which is
	// inconsistent but unlikely to change. That means we don't REALLY need to
	// handle it here, but let's be extra careful and extract the most concrete
	// type possible.
	//FIXME: map?
	if k := util.NonPointer(util.NativeType(context.Type)).Kind; k != types.Struct && k != types.Slice {
		return result, nil
	}

	unions := utfv.shared[context.Path.String()]
	if len(unions) == 0 {
		return result, nil
	}

	// Sort the keys for stable output.
	keys := make([]string, 0, len(unions))
	for k := range unions {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	for _, unionName := range keys {
		u := unions[unionName]
		if len(u.fieldMembers) > 0 || u.discriminator != nil || len(u.itemMatchers) > 0 {
			// TODO: Avoid the "local" here. This was added to to avoid errors caused when the package is an empty string.
			//       The correct package would be the output package but is not known here. This does not show up in generated code.
			// TODO: Append a consistent hash suffix to avoid generated name conflicts?
			//FIXME: name needed for this based on path?
			supportVarName := PrivateVar{Name: "UnionMembershipFor" + sanitizeName(context.Type.Name.Name+unionName), Package: "local"}

			var extractorArgs []any
			ptrType := types.PointerTo(context.Type)

			// Handle regular field-based unions
			for _, member := range u.fieldMembers {
				extractor := FunctionLiteral{
					Parameters: []ParamResult{{Name: "obj", Type: ptrType}},
					Results:    []ParamResult{{Type: types.Bool}},
				}
				nt := util.NativeType(member.Type)
				switch nt.Kind {
				case types.Pointer, types.Map, types.Slice:
					extractor.Body = fmt.Sprintf("if obj == nil {return false}; return obj.%s != nil", member.Name)
				case types.Builtin:
					extractor.Body = fmt.Sprintf("if obj == nil {return false}; var z %s; return obj.%s != z", member.Type, member.Name)
				default:
					// This should be caught before we get here, but JIC.
					return Validations{}, fmt.Errorf("unsupported union member kind: %s", nt.Kind)
				}
				extractorArgs = append(extractorArgs, extractor)
			}

			// Handle item-based unions for lists
			if context.Type.Kind == types.Slice && len(u.itemMatchers) > 0 {
				elemType := util.NonPointer(context.Type.Elem)
				// Sort matcher paths for stable output
				matcherPaths := make([]string, 0, len(u.itemMatchers))
				for path := range u.itemMatchers {
					matcherPaths = append(matcherPaths, path)
				}
				// No sort on matcherPaths to preserve tag order

				for _, fullPath := range matcherPaths {
					matcherData := u.itemMatchers[fullPath]
					matcher := matcherData

					// Build matcher conditions
					var conditions []string
					for key, value := range matcher {
						member := util.GetMemberByJSON(elemType, key)
						if member == nil {
							return Validations{}, fmt.Errorf("no field with JSON name %q for matcher", key)
						}
						var condition string
						switch v := value.(type) {
						case string:
							condition = fmt.Sprintf("item.%s == %q", member.Name, v)
						case int:
							condition = fmt.Sprintf("item.%s == %d", member.Name, v)
						case bool:
							condition = fmt.Sprintf("item.%s == %t", member.Name, v)
						default:
							condition = fmt.Sprintf("item.%s == %v", member.Name, v)
						}
						conditions = append(conditions, condition)
					}

					// Generate extractor that wraps SliceItem
					extractor := FunctionLiteral{
						Parameters: []ParamResult{{Name: "list", Type: context.Type}},
						Results:    []ParamResult{{Type: types.Bool}},
					}
					extractor.Body = fmt.Sprintf(`
var matched *%s
validate.SliceItem(ctx, op, fldPath, list, nil,  // No oldList; Union ratchets bool
    func(item *%s) bool { return %s },  // Matcher
    func(ctx context.Context, op operation.Operation, itemPath *field.Path, newItem, oldItem *%s) field.ErrorList {
        matched = newItem
        return nil
    },
)
return matched != nil && matched.State != ""  // Set check (customize as needed)
`, elemType.Name.Name, elemType.Name.Name, strings.Join(conditions, " && "), elemType.Name.Name)

					extractorArgs = append(extractorArgs, extractor)
				}
			}

			if u.discriminator != nil {
				supportVar := Variable(supportVarName,
					Function(unionMemberTagName, DefaultFlags, newDiscriminatedUnionMembership,
						append([]any{*u.discriminator}, toSliceAny(getDisplayFields(u, context))...)...))
				result.Variables = append(result.Variables, supportVar)

				discriminatorExtractor := FunctionLiteral{
					Parameters: []ParamResult{{Name: "obj", Type: ptrType}},
					Results:    []ParamResult{{Type: types.String}},
					Body:       fmt.Sprintf("if obj == nil {return \"\"}; return string(obj.%s)", u.discriminatorMember.Name), // Cast to string
				}

				extraArgs := append([]any{supportVarName, discriminatorExtractor}, extractorArgs...)
				fn := Function(unionMemberTagName, DefaultFlags, discriminatedUnionValidator, extraArgs...)
				result.Functions = append(result.Functions, fn)
			} else {
				supportVar := Variable(supportVarName, Function(unionMemberTagName, DefaultFlags, newUnionMembership, toSliceAny(getDisplayFields(u, context))...))
				result.Variables = append(result.Variables, supportVar)

				extraArgs := append([]any{supportVarName}, extractorArgs...)
				fn := Function(unionMemberTagName, DefaultFlags, unionValidator, extraArgs...)
				result.Functions = append(result.Functions, fn)
			}
		}
	}

	return result, nil
}

func getDisplayFields(u *union, context Context) [][2]string {
	displayFields := make([][2]string, len(u.fields))
	listFieldName := context.Path.String()
	pathParts := strings.Split(listFieldName, ".")
	if len(pathParts) > 0 {
		listFieldName = pathParts[len(pathParts)-1]
	}
	for i, f := range u.fields {
		fieldName := f[0]
		memberName := f[1]
		if matcher, isItem := u.itemMatchers[fieldName]; isItem {
			// Clean matcher to name="value"
			var matcherParts []string
			for k, v := range matcher {
				valStr := fmt.Sprint(v)
				if str, ok := v.(string); ok {
					valStr = fmt.Sprintf("\"%s\"", str)
				}
				matcherParts = append(matcherParts, fmt.Sprintf("%s=%s", k, valStr))
			}
			fieldName = fmt.Sprintf("%s[%s]", listFieldName, strings.Join(matcherParts, ", "))
		} else {
			// Regular: Keep as is or prefix if needed
		}
		displayFields[i] = [2]string{fieldName, memberName}
	}
	return displayFields
}

func toSliceAny[T any](t []T) []any {
	result := make([]any, len(t))
	for i, v := range t {
		result[i] = v
	}
	return result
}

func sanitizeName(name string) string {
	// Replace invalid chars with _
	re := regexp.MustCompile(`[^a-zA-Z0-9_]`)
	return re.ReplaceAllString(name, "_")
}

const (
	unionDiscriminatorTagName = "k8s:unionDiscriminator"
	unionMemberTagName        = "k8s:unionMember"
)

type unionDiscriminatorTagValidator struct {
	//FIXME: document key
	shared map[string]unions
}

func (unionDiscriminatorTagValidator) Init(_ Config) {}

func (unionDiscriminatorTagValidator) TagName() string {
	return unionDiscriminatorTagName
}

// Shared between unionDiscriminatorTagValidator and unionMemberTagValidator.
var unionTagValidScopes = sets.New(ScopeField, ScopeListVal)

func (unionDiscriminatorTagValidator) ValidScopes() sets.Set[Scope] {
	return unionTagValidScopes
}

func (udtv unionDiscriminatorTagValidator) GetValidations(context Context, tag codetags.Tag) (Validations, error) {
	// This tag can apply to value and pointer fields, as well as typedefs
	// (which should never be pointers). We need to check the concrete type.
	if t := util.NonPointer(util.NativeType(context.Type)); t != types.String {
		return Validations{}, fmt.Errorf("can only be used on string types (%s)", rootTypeString(context.Type, t))
	}
	pp := context.ParentPath.String()
	if udtv.shared[pp] == nil {
		udtv.shared[pp] = unions{}
	}
	unionArg, _ := tag.NamedArg("union") // optional
	u := udtv.shared[pp].getOrCreate(unionArg.Value)

	var discriminatorFieldName string
	if jsonAnnotation, ok := tags.LookupJSON(*context.Member); ok {
		discriminatorFieldName = jsonAnnotation.Name
		u.discriminator = &discriminatorFieldName
		u.discriminatorMember = context.Member
	}

	// This tag does not actually emit any validations, it just accumulates
	// information. The validation is done by the unionTypeOrFieldValidator.
	return Validations{}, nil
}

func (udtv unionDiscriminatorTagValidator) Docs() TagDoc {
	return TagDoc{
		Tag:         udtv.TagName(),
		Scopes:      udtv.ValidScopes().UnsortedList(),
		Description: "Indicates that this field is the discriminator for a union.",
		Args: []TagArgDoc{{
			Name:        "union",
			Description: "<string>",
			Docs:        "the name of the union, if more than one exists",
			Type:        codetags.ArgTypeString,
		}},
	}
}

type unionMemberTagValidator struct {
	shared map[string]unions
}

func (unionMemberTagValidator) Init(_ Config) {}

func (unionMemberTagValidator) TagName() string {
	return unionMemberTagName
}

func (unionMemberTagValidator) ValidScopes() sets.Set[Scope] {
	return unionTagValidScopes
}

func (umtv unionMemberTagValidator) GetValidations(context Context, tag codetags.Tag) (Validations, error) {
	var fieldName string
	var pp string
	var unionArg codetags.Arg

	pp = context.ParentPath.String()
	unionArg, _ = tag.NamedArg("union") // optional

	if context.Scope == ScopeListVal {
		// Virtual for list val: Use path as name, context.Type for nt
		if context.Path == nil {
			return Validations{}, fmt.Errorf("no path for list val union member")
		}
		fieldName = context.Path.String() // e.g., "Tasks[name=\"succeeded\"]"
	} else {
		jsonTag, ok := tags.LookupJSON(*context.Member)
		if !ok {
			return Validations{}, fmt.Errorf("field %q is a union member but has no JSON struct field tag", context.Member)
		}
		fieldName = jsonTag.Name
		if len(fieldName) == 0 {
			return Validations{}, fmt.Errorf("field %q is a union member but has no JSON name", context.Member)
		}
	}

	if umtv.shared[pp] == nil {
		umtv.shared[pp] = unions{}
	}

	var memberName string
	if memberNameArg, ok := tag.NamedArg("memberName"); ok { // optional
		memberName = memberNameArg.Value
	} else if context.Scope != ScopeListVal {
		memberName = context.Member.Name // default
	}

	u := umtv.shared[pp].getOrCreate(unionArg.Value)
	u.fields = append(u.fields, [2]string{fieldName, memberName})

	if context.Scope == ScopeListVal {
		// Extract matcher criteria from the path
		// Path looks like: "Pipeline.Tasks[{"name": "succeeded"}]"
		matcher, err := extractMatcherFromPath(fieldName)
		if err != nil {
			return Validations{}, fmt.Errorf("failed to extract matcher from path %s: %w", fieldName, err)
		}
		u.itemMatchers[fieldName] = matcher
	} else {
		u.fieldMembers = append(u.fieldMembers, context.Member)
	}

	// This tag does not actually emit any validations, it just accumulates
	// information. The validation is done by the unionTypeOrFieldValidator.
	return Validations{}, nil
}

func (umtv unionMemberTagValidator) Docs() TagDoc {
	return TagDoc{
		Tag:         umtv.TagName(),
		Scopes:      umtv.ValidScopes().UnsortedList(),
		Description: "Indicates that this field is a member of a union.",
		Args: []TagArgDoc{{
			Name:        "union",
			Description: "<string>",
			Docs:        "the name of the union, if more than one exists",
			Type:        codetags.ArgTypeString,
		}, {
			Name:        "memberName",
			Description: "<string>",
			Docs:        "the discriminator value for this member",
			Default:     "the field's name",
			Type:        codetags.ArgTypeString,
		}},
	}
}

// extractMatcherFromPath extracts the matcher criteria from a path like "Pipeline.Tasks[{"name": "succeeded"}]"
func extractMatcherFromPath(path string) (map[string]any, error) {
	// Find the JSON object within the square brackets
	re := regexp.MustCompile(`\[({.*?})\]`)
	matches := re.FindStringSubmatch(path)
	if len(matches) < 2 {
		return nil, fmt.Errorf("no matcher criteria found in path")
	}

	// Parse the JSON
	var matcher map[string]any
	if err := json.Unmarshal([]byte(matches[1]), &matcher); err != nil {
		return nil, fmt.Errorf("failed to parse matcher JSON: %w", err)
	}
	return matcher, nil
}

// union defines how a union validation will be generated, based
// on +k8s:unionMember and +k8s:unionDiscriminator tags found in a go struct.
type union struct {
	// fields provides field information about all the members of the union.
	// Each item provides a fieldName and memberName pair, where [0] identifies
	// the field name and [1] identifies the union member Name. fields is index
	// aligned with fieldMembers.
	// If member name is not set, it defaults to the go struct field name.
	fields [][2]string
	// fieldMembers describes all the members of the union.
	fieldMembers []*types.Member

	// discriminator is the name of the discriminator field
	discriminator *string
	// discriminatorMember describes the discriminator field.
	discriminatorMember *types.Member

	// itemMatchers stores matcher criteria for item-based unions
	// Key is the virtual path (fieldName), value is the matcher map
	itemMatchers map[string]map[string]any
}

// unions represents all the unions for a go struct.
type unions map[string]*union

// getOrCreate gets a union by name, or initializes a new union by the given name.
func (us unions) getOrCreate(name string) *union {
	var u *union
	var ok bool
	if u, ok = us[name]; !ok {
		u = &union{
			itemMatchers: make(map[string]map[string]any),
		}
		us[name] = u
	}
	return u
}

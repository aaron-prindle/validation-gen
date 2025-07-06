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
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/code-generator/cmd/validation-gen/util"
	"k8s.io/gengo/v2"
	"k8s.io/gengo/v2/codetags"
	"k8s.io/gengo/v2/types"
)

const (
	updateTagName = "k8s:update"

	// Update constraints
	constraintNoSet        = "NoSet"
	constraintNoUnset      = "NoUnset"
	constraintNoModify     = "NoModify"
	constraintNoAddItem    = "NoAddItem"
	constraintNoRemoveItem = "NoRemoveItem"
)

func init() {
	shared := map[string]*updateFieldMetadata{}
	RegisterFieldValidator(updateFieldValidator{byFieldPath: shared})
	RegisterTagValidator(updateTagCollector{byFieldPath: shared})
}

type updateFieldMetadata struct {
	updateConstraints []string // NoSet, NoUnset, NoModify, etc.

	// List metadata (gathered from listType and listMapKey tags)
	listType    string   // "map", "set", or ""
	listMapKeys []string // field names for map keys
}

func ensureUpdateFieldMetadata(byFieldPath map[string]*updateFieldMetadata, path string) *updateFieldMetadata {
	if fm, ok := byFieldPath[path]; ok {
		return fm
	}
	fm := &updateFieldMetadata{}
	byFieldPath[path] = fm
	return fm
}

// updateTagCollector collects +k8s:update tags
type updateTagCollector struct {
	byFieldPath map[string]*updateFieldMetadata
}

func (updateTagCollector) Init(_ Config) {}

func (updateTagCollector) TagName() string {
	return updateTagName
}

var updateTagValidScopes = sets.New(ScopeField, ScopeMapVal, ScopeListVal)

func (updateTagCollector) ValidScopes() sets.Set[Scope] {
	return updateTagValidScopes
}

func (utc updateTagCollector) GetValidations(context Context, tag codetags.Tag) (Validations, error) {
	fm := ensureUpdateFieldMetadata(utc.byFieldPath, context.Path.String())

	// Parse the update constraints
	tagValue := strings.TrimSpace(tag.Value)
	tagValue = strings.Trim(tagValue, "`\"")

	if tagValue == "" {
		return Validations{}, nil
	}

	// Split and clean constraints
	for _, value := range strings.Split(tagValue, ",") {
		if constraint := strings.TrimSpace(value); constraint != "" {
			fm.updateConstraints = append(fm.updateConstraints, constraint)
		}
	}

	// Don't generate validations here, just collect
	return Validations{}, nil
}

func (utc updateTagCollector) Docs() TagDoc {
	return TagDoc{
		Tag:          utc.TagName(),
		Scopes:       utc.ValidScopes().UnsortedList(),
		PayloadsType: codetags.ValueTypeString,
		Description: "Provides fine-grained control over field update operations. " +
			"Supported values: NoSet (prevents unset→set), NoUnset (prevents set→unset), " +
			"NoModify (prevents value changes), NoAddItem (prevents adding to lists/maps), " +
			"NoRemoveItem (prevents removing from lists/maps). " +
			"Multiple values can be specified separated by commas. " +
			"Examples: +k8s:update=`NoModify,NoUnset` for set-once fields; " +
			"+k8s:update=`NoRemoveItem` for append-only lists; " +
			"+k8s:update=`NoSet` for fields that must be set at creation or never.",
	}
}

// updateFieldValidator processes all collected metadata and generates validations
type updateFieldValidator struct {
	byFieldPath map[string]*updateFieldMetadata
}

func (updateFieldValidator) Init(_ Config) {}

func (updateFieldValidator) Name() string {
	return "updateFieldValidator"
}

var (
	noSetValidator            = types.Name{Package: libValidationPkg, Name: "NoSet"}
	noUnsetValidator          = types.Name{Package: libValidationPkg, Name: "NoUnset"}
	noModifyValidator         = types.Name{Package: libValidationPkg, Name: "NoModify"}
	noAddItemValidator        = types.Name{Package: libValidationPkg, Name: "NoAddItem"}
	noRemoveItemValidator     = types.Name{Package: libValidationPkg, Name: "NoRemoveItem"}
	noAddItemMapValidator     = types.Name{Package: libValidationPkg, Name: "NoAddItemMap"}
	noRemoveItemMapValidator  = types.Name{Package: libValidationPkg, Name: "NoRemoveItemMap"}
	noRemoveLastItemValidator = types.Name{Package: libValidationPkg, Name: "NoRemoveLastItem"}
)

func (ufv updateFieldValidator) GetValidations(context Context) (Validations, error) {
	fm := ufv.byFieldPath[context.Path.String()]
	if fm == nil {
		fm = ensureUpdateFieldMetadata(ufv.byFieldPath, context.Path.String())
	}

	// Collect list metadata if present
	if err := ufv.collectListMetadata(context, fm); err != nil {
		return Validations{}, err
	}

	// If no update constraints, nothing to do
	if len(fm.updateConstraints) == 0 {
		return Validations{}, nil
	}

	// Check field requiredness
	fieldInfo, err := ufv.getFieldInfo(context)
	if err != nil {
		return Validations{}, err
	}

	// Calculate update capabilities
	updateCaps := ufv.calculateUpdateCapabilities(fieldInfo, fm.updateConstraints)

	// Generate validations
	return ufv.generateValidations(context, fieldInfo, updateCaps)
}

// fieldInfo holds information about field tags
type fieldInfo struct {
	hasRequired bool
	hasOptional bool
	hasDefault  bool
}

// updateCapabilities represents what update operations are allowed
type updateCapabilities struct {
	canSet        bool
	canUnset      bool
	canModify     bool
	canAddItem    bool
	canRemoveItem bool
}

func (ufv updateFieldValidator) collectListMetadata(context Context, fm *updateFieldMetadata) error {
	if context.Member == nil {
		return nil
	}

	listTags, err := gengo.ExtractFunctionStyleCommentTags("+", []string{"listType", "listMapKey"}, context.Member.CommentLines)
	if err != nil {
		return fmt.Errorf("failed to read list tags: %w", err)
	}

	if listTypeTags, ok := listTags["listType"]; ok && len(listTypeTags) > 0 {
		fm.listType = listTypeTags[0].Value
	}

	if listMapKeyTags, ok := listTags["listMapKey"]; ok {
		for _, tag := range listMapKeyTags {
			fm.listMapKeys = append(fm.listMapKeys, tag.Value)
		}
	}

	return nil
}

func (ufv updateFieldValidator) getFieldInfo(context Context) (fieldInfo, error) {
	info := fieldInfo{}

	if context.Member == nil {
		return info, nil
	}

	tagsByName, err := gengo.ExtractFunctionStyleCommentTags("+", []string{
		requiredTagName,
		optionalTagName,
		defaultTagName,
	}, context.Member.CommentLines)
	if err != nil {
		return info, fmt.Errorf("failed to read tags: %w", err)
	}

	_, info.hasRequired = tagsByName[requiredTagName]
	_, info.hasOptional = tagsByName[optionalTagName]
	_, info.hasDefault = tagsByName[defaultTagName]

	// Validate tag combinations
	if info.hasRequired && info.hasOptional {
		return info, fmt.Errorf("field cannot have both +k8s:required and +k8s:optional")
	}

	return info, nil
}

func (ufv updateFieldValidator) calculateUpdateCapabilities(info fieldInfo, constraints []string) updateCapabilities {
	// Start with all capabilities enabled
	caps := updateCapabilities{
		canSet:        true,
		canUnset:      true,
		canModify:     true,
		canAddItem:    true,
		canRemoveItem: true,
	}

	// Apply rules based on field tags
	if info.hasRequired {
		caps.canSet = false
		caps.canUnset = false
	}

	if info.hasOptional && info.hasDefault {
		// optional + default = effectively required
		caps.canSet = false
		caps.canUnset = false
	}

	// Apply explicit constraints
	for _, constraint := range constraints {
		switch constraint {
		case constraintNoSet:
			caps.canSet = false
		case constraintNoUnset:
			caps.canUnset = false
		case constraintNoModify:
			caps.canModify = false
		case constraintNoAddItem:
			caps.canAddItem = false
		case constraintNoRemoveItem:
			caps.canRemoveItem = false
		}
	}

	return caps
}

func (ufv updateFieldValidator) generateValidations(
	context Context,
	info fieldInfo,
	caps updateCapabilities,
) (Validations, error) {
	var result Validations

	// IMPORTANT: Use ShortCircuit flag so these run in the same group as +k8s:optional
	// This ensures that both validators get a chance to run before the early return decision

	// Add basic validation functions
	if !caps.canSet {
		result.AddFunction(Function("update:NoSet", ShortCircuit, noSetValidator))
	}
	if !caps.canUnset {
		result.AddFunction(Function("update:NoUnset", ShortCircuit, noUnsetValidator))
	}
	if !caps.canModify {
		result.AddFunction(Function("update:NoModify", ShortCircuit, noModifyValidator))
	}

	// Handle compound types (lists and maps)
	t := util.NonPointer(util.NativeType(context.Type))
	switch t.Kind {
	case types.Slice:
		ufv.addSliceValidations(&result, caps, info.hasRequired)
	case types.Map:
		ufv.addMapValidations(&result, caps)
	}

	return result, nil
}

func (ufv updateFieldValidator) addSliceValidations(result *Validations, caps updateCapabilities, hasRequired bool) {
	// Also use ShortCircuit for list/map validations
	if !caps.canAddItem {
		result.AddFunction(Function("update:NoAddItem", ShortCircuit, noAddItemValidator))
	}

	if !caps.canRemoveItem {
		result.AddFunction(Function("update:NoRemoveItem", ShortCircuit, noRemoveItemValidator))
	} else if hasRequired {
		// Special handling for required lists - cannot remove last item
		result.AddFunction(Function("update:NoRemoveLastItem", ShortCircuit, noRemoveLastItemValidator))
	}
}

func (ufv updateFieldValidator) addMapValidations(result *Validations, caps updateCapabilities) {
	// Also use ShortCircuit for map validations
	if !caps.canAddItem {
		result.AddFunction(Function("update:NoAddItem", ShortCircuit, noAddItemMapValidator))
	}
	if !caps.canRemoveItem {
		result.AddFunction(Function("update:NoRemoveItem", ShortCircuit, noRemoveItemMapValidator))
	}
}

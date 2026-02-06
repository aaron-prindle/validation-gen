# Declarative Validation Migration Guide

This document is a guide for migrating existing hand-written validation logic in Kubernetes API types to the new **Declarative Validation** system using `+k8s:` tags.

The goal when migrating hand-written validation is to create migration PRs that allow for strict backward compatibility with existing error messages/behaviours. For these migrations, the DV framework operates in **Shadow Mode** by default. This means the declarative tags are evaluated alongside the handwritten code, and any mismatches are recorded as metrics, but the declarative errors do not reject the request (the handwritten code remains authoritative). In the future, once stability is proven via metrics, the related hand-written code will be removed.

---

## Migration PR Commit Structure

Maintainers strongly recommend the following **commit strategy** to ensure reviews are smooth and debugging is easy.

### 1. **Commit #1: Infrastructure Setup (If not already wired)**
-   Wire up `doc.go` (enable generation).
-   Update `strategy.go` to call the declarative validator.
-   Create the `declarative_validation_test.go` file with the test harness.
-   Add the initial (empty) generated files.
-   **Goal**: Prove the plumbing works without changing any logic.

### 2. **Commit #2: The First Tag/Field**
-   Pick a **single, simple field** (e.g., a top-level `Required` field).
-   Add a **single DV tag** (e.g., `+k8s:required`).
-   Mark the hand-written error as covered.
-   Add precise test cases.
    *   *Emphasis on test-cases*: We want enough to prove that a field is correct, but we don't want to duplicate all the per-tag tests (e.g. a `k8s-short-name` field has dozens of corner cases, but we really only need a few). We want each case to be as short and specific as possible, which is why the "tweak" pattern and the use of `.WithOrigin()` for Invalid() errors is so important. Specify as little as possible to be sufficiently precise.
-   **Goal**: Establish the pattern for the rest of the PR.

### 3. **Commits #3...N: Iterate on the Same Field**
-   Use additional DV tags on the same field, **one tag per commit**.
-   Repeat until that field is fully migrated or all that is left is not handled by DV yet.
-   Includes generated code and tests for each tag addition.

### 4. **Commits N+1...M: Iterate on More Fields**
-   Repeat the above process for additional fields.
-   **Rule of Thumb**: The emphasis is on lots of small commits. Maintaining commit discipline is hard but worthwhile.

*Note: If DV exposes bugs in the existing testing or the hand-written validation itself, those should be fixed as a PR of its own or as commits at the front of the PR.*

---

## Prerequisites & Setup

### 1. Enable Code Generation
In `pkg/apis/<group>/<version>/doc.go`, enable the generator if not already enabled for that type:

```go
// +k8s:validation-gen=TypeMeta
// +k8s:validation-gen-input=k8s.io/api/<group>/<version>
package v1
```

### 2. Update Strategy
In `pkg/registry/<group>/<resource>/strategy.go`, modify the `Validate` and `ValidateUpdate` methods to use `ValidateDeclarativelyWithMigrationChecks`.

```go
func (strategy) Validate(ctx context.Context, obj runtime.Object) field.ErrorList {
    // ... setup ...
    // OLD: return validation.ValidateFoo(obj.(*api.Foo))
    
    // NEW:
    return rest.ValidateDeclarativelyWithMigrationChecks(
        ctx,
        legacyscheme.Scheme,
        obj,
        nil, // Old object (for updates)
        validation.ValidateFoo(), // Pass the old validation function errors
        operation.Create,
    )
}
```

### 3. Add Equivalence Tests
Create `declarative_validation_test.go` in the registry directory:

```go
func TestDeclarativeValidation(t *testing.T) {
    apitesting.VerifyValidationEquivalence(t,
        &api.Foo{},
        func(obj runtime.Object) field.ErrorList {
            return strategy.Validate(context.TODO(), obj)
        },
    )
}
```

---

## Common Pitfalls & Friction Points

### 1. Short-Circuiting Behavior
**The Issue:**
Declarative Validation (DV) tags like `+k8s:optional`, `+k8s:required`, `+k8s:immutable`, and `+k8s:maxItems` are **short-circuiting**.
-   If a field is missing (and not required), validation stops there.
-   If a field is immutable and changed, validation stops there.

**Hand-written Mismatch:**
Hand-written code often checks *everything*.
-   *Hand-written*: Returns "Required" error AND "Invalid Format" error for an empty string.
-   *DV*: Returns only "Required".

**Resolution:**
You must identify if the hand-written code runs multiple checks that DV would skip.
-   **Check Parent Fields**: If you migrate `spec.template.spec.containers`, look at parents. Is `spec` optional? Is `template` immutable? If so, you must migrate those short-circuiting checks first or replicate the logic.

### 2. Duplicate Errors
**The Issue:**
Sometimes hand-written validation accidentally produces duplicate errors (e.g., checking "Required" in two places). DV will strictly produce one.

**Resolution:**
-   **Fix Hand-written First**: Submit a separate PR to deduplicate errors in the hand-written code.
-   **Then Migrate**: Once clean, migrate to DV.

### 3. Shared Validation Helpers
**The Issue:**
A helper function `ValidateCommonSpec` might be used by multiple types (e.g., `Type1` and `Type2`).
-   If you migrate `Type1` and add `.MarkCoveredByDeclarative()` to the helper, you also have marked `Type2` as declarative (which isn't migrated yet -> BAD).

**Resolution: The Options Pattern**
Refactor the helper to accept options.

**Example:**

```go
// validation.go

type ValidateCSIDriverNameOption int

const (
    // Explicitly identify which checks are covered
    RequiredCovered ValidateCSIDriverNameOption = iota
    SizeCovered
)

// Add variadic options
func ValidateCSIDriverName(name string, fldPath *field.Path, opts ...ValidateCSIDriverNameOption) field.ErrorList {
    allErrs := field.ErrorList{}

    if len(name) == 0 {
        err := field.Required(fldPath, "")
        // Only mark covered if the caller says so
        if slices.Contains(opts, RequiredCovered) {
            err = err.MarkCoveredByDeclarative()
        }
        allErrs = append(allErrs, err)
        return allErrs // Short-circuit to match DV behavior
    }
    // ...
}
```

**Usage in validations.go:**
```go
// For the migrated type:
ValidateCSIDriverName(name, path, validation.RequiredCovered)

// For the unmigrated type:
ValidateCSIDriverName(name, path) // No options, errors returned normally
```

### 4. Renamed or Moved Fields (Path Normalization)
**The Issue:**
Declarative Validation (DV) generated code validates the **specific versioned API types** (e.g., `v1beta1` structs). However, hand-written validation typically runs against the kubernetes **internal type** for that type.  NOTE: this **internal type** usually maps to the **latest version** of the type.

If a field structure changed between versions (e.g., nesting a field that was previously top-level), the error paths will differ:
-   **Declarative Validation (v1beta1)**: Reports error at `spec.oldField` (the path in `v1beta1`).
-   **Hand-written Validation**: Reports error at `spec.newNested.oldField` (the path in the latest version).

This causes a mismatch in the migration check (`spec.oldField` != `spec.newNested.oldField`), even though the validation logic is equivalent.

**Resolution:**
Use **Normalization Rules** in `strategy.go`. These rules allow the migration checker to transform the path from one format (the versioned path) to the other (the latest path) before comparing them.

**Example:**
In `pkg/registry/resource/resourceclaim/strategy.go`, the `v1beta1` version had fields directly under `requests`, but the latest version moved them under `requests[i].exactly`.

```go
var resourceClaimNormalizationRules = []field.NormalizationRule{
    {
        // Map v1beta1 path (spec.devices.requests[i].selectors) 
        // to latest path (spec.devices.requests[i].exactly.selectors)
        Regexp:      regexp.MustCompile(`spec\.devices\.requests\[(\d+)\]\.selectors`),
        Replacement: "spec.devices.requests[$1].exactly.selectors",
    },
}

func (s *resourceclaimStrategy) Validate(ctx context.Context, obj runtime.Object) field.ErrorList {
    // ...
    return rest.ValidateDeclarativelyWithMigrationChecks(
        ctx, 
        legacyscheme.Scheme, 
        claim, 
        nil, 
        allErrs, 
        operation.Create, 
        // Pass the rules here:
        rest.WithNormalizationRules(resourceClaimNormalizationRules),
    )
}
```

**How it works:**
The error matcher applies the regex to the Declarative Validation error paths. If `spec.devices.requests[0].selectors` matches, it transforms it to `spec.devices.requests[0].exactly.selectors`. If this transformed path matches the hand-written error path, the migration check passes.

---

## Example PRs
- https://github.com/kubernetes/kubernetes/pull/134796 (Storage: wire storage group for declarative validation and +k8s:required to StorageClass.Provisioner)
- https://github.com/kubernetes/kubernetes/pull/135520 (Core: Enable Declarative Validation for LimitRange without plumbing DV through type)
- https://github.com/kubernetes/kubernetes/pull/135935 (Networking: Network Tag CIDR Declarative Validation)
- https://github.com/kubernetes/kubernetes/pull/135046 (Node: wire node group for declarative validation)
- https://github.com/kubernetes/kubernetes/pull/134909 (Network: wire network group for declarative validation and +k8s:required to IngressClassParametersReference.Name)
- https://github.com/kubernetes/kubernetes/pull/135412 (Autoscaling: HPA: Enable DV support for MaxReplicas)
- https://github.com/kubernetes/kubernetes/pull/135890 (Discovery: wire discovery group for declarative validation and migrate Endpoint.Addresses + EndpointSlice.AddressType)

## Step-by-Step Migration Example

### Phase 1: Analyze
1.  Choose a field: `spec.driverName`.
2.  Check `types.go`: It is a string.
3.  Check `validation.go`:
    ```go
    if len(spec.DriverName) == 0 {
        allErrs = append(allErrs, field.Required(...))
    }
    ```
4. For fields that have additional parent fields in the field path (spec.foo.bar.baz), for each parent you MUST ADD any short-circuit validation tags associated with the hand-written validation for the parent.  These include: +k8s:optional, +k8s:required, and +k8s:immutable/+k8s:update. This involves going through the hand-written validation.go logic for the parent fields and seeing if they have logic related to the list of short-circuit validation tags above (eg: is field foo required?  is it immutable?  if so need to add +k8s: required and +k8s:immutable to it)


### Phase 2: Apply Tag
In `types.go`:
```go
type CSIDriverSpec struct {
    // +k8s:required
    DriverName string `json:"driverName"`
}
```

### Phase 3: Update Hand-written Code
In `validation.go`:
```go
if len(spec.DriverName) == 0 {
    // Mark this specific error as covered
    allErrs = append(allErrs, field.Required(...).MarkCoveredByDeclarative())
}
```

### Phase 4: Generate & Test
1.  `hack/update-codegen.sh validation`
2.  `go test ./pkg/registry/...`

If `VerifyValidationEquivalence` passes, the migration for that field is correct.

---

## FAQ

**Q: A field was renamed/moved between `v1beta1` and `v1`. How can I use DV to handle this?**
A: DV validation runs on the versioned type. If your hand-written validation outputs errors using new paths, you must use **Path Normalization** to map them. See [Pitfall #4](#4-renamed-or-moved-fields-path-normalization).

**Q: My test failed with "Validation mismatch".**
A: Look at the diff.
-   If DV is missing an error: Did you forget a tag?
-   Is the tag correct? Check for existing occurrences of the same tag.
-   If Hand-written has an extra error: Is DV short-circuiting? (See Pitfall #1).
-   If DV has an extra error: Did you forget to `.MarkCoveredByDeclarative()`?
-   If Origin Mismatch -> Add origin in the handwritten validation code

**Q: Is there a way to add a validation that DV doesn't support yet.**
A: Depending on the situation you can create a new DV tag under validation-gen.  Currently this is has not been done by by someone outside of the direct Declarative Validation working group but is possible for simpler cases where the API is likely straightforward and the implementation is simple.  For example, possibly adding a new format to +k8s:format=<...>.
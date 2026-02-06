# API Reviewers Guide for Declarative Validation

Welcome to the API Reviewers guide for Declarative Validation (DV). Starting in v1.36, Declarative Validation is the recommended way to author API validation logic for Kubernetes.

As an API reviewer, your goal is to ensure APIs are robust, consistent, and maintainable. DV helps achieve this by moving validation logic out of procedural Go code (`validation.go`) and into declarative comment tags directly on the API types (`types.go`).

This guide focuses on the practical aspects of reviewing PRs that use DV.

---

## 1. The Happy Path: What to Expect

When reviewing a PR, you will generally encounter one of two scenarios:
1.  **New APIs/Fields**: Using DV as the authoritative source of truth from Day 1.
2.  **Migrations**: Moving existing handwritten validation to DV tags.

Before reviewing the logic, familiarize yourself with the [Official Declarative Validation Tag Catalog](https://kubernetes.io/docs/reference/using-api/declarative-validation/). This is your reference for what tags exist, what they do, and their stability level.

### Scenario A: New API Field (Authoritative DV)

When a developer adds a new field and wants to use DV, the tags in `types.go` are the *only* validation logic. There should be no fallback handwritten code for these standard rules.

**1. `types.go` (The Single Source of Truth)**
The developer adds standard tags directly to the new field.

```go
type MyNewFeatureSpec struct {
    // +required
    // +k8s:required
    // +k8s:maxLength=256
    // +k8s:format=k8s-short-name
    FeatureName string `json:"featureName"`
}
```

**2. `strategy.go` (The Plumbing)**
For new APIs to use these tags authoritatively, the strategy must be explicitly told to enforce them using `rest.WithDeclarativeEnforcement()`.

```go
func (myStrategy) Validate(ctx context.Context, obj runtime.Object) field.ErrorList {
    allErrs := validation.ValidateMyFeature(obj.(*myapi.MyFeature)) // Any complex handwritten rules
    
    return rest.ValidateDeclarativelyWithMigrationChecks(
        ctx, 
        legacyscheme.Scheme, 
        obj, 
        nil, 
        allErrs, 
        operation.Create, 
        rest.WithDeclarativeEnforcement(), // <--- Critical for New APIs
    )
}
```

**3. `validation_test.go` (The Tests)**
Tests for DV rely on marking expected errors to confirm they came from the DV framework, not handwritten code. Because `WithDeclarativeEnforcement()` is used, errors from standard tags are marked as "Non-Shadowed" (meaning they are actively enforced).

```go
"feature name too long": {
    obj: mkFeature(func(f *myapi.MyFeature) {
        f.Spec.FeatureName = strings.Repeat("a", 257)
    }),
    expectedErrs: field.ErrorList{
        field.TooLongMaxLength(field.NewPath("spec", "featureName"), 257, 256).MarkNonShadowed(),
    },
},
```

### Scenario B: Migrating Existing Validation

When migrating *existing* handwritten code, the goal is strict backward compatibility. The DV framework runs the tags alongside the handwritten code, compares the results, and emits metrics if they differ. The declarative errors are *suppressed* (shadowed) by default, and the handwritten errors are returned to the user.

**1. `types.go`**
Add the appropriate tag that matches the handwritten logic.

```go
type LegacyFeatureSpec struct {
    // +k8s:minimum=1
    Replicas *int32 `json:"replicas,omitempty"`
}
```

**2. `validation.go` (Marking Coverage)**
The old handwritten error *must* be explicitly marked as covered by the new tag. This tells the migration checker "Yes, I know DV handles this now."

```go
if spec.Replicas != nil && *spec.Replicas < 1 {
    // Must append .MarkCoveredByDeclarative()
    allErrs = append(allErrs, field.Invalid(fldPath, *spec.Replicas, "must be greater than or equal to 1").MarkCoveredByDeclarative())
}
```

**3. `strategy.go` (The Plumbing)**
The strategy calls the DV runner, but *without* the enforcement flag. This puts it in "Implicit Shadow" mode.

```go
func (myStrategy) Validate(ctx context.Context, obj runtime.Object) field.ErrorList {
    // ...
    return rest.ValidateDeclarativelyWithMigrationChecks(
        ctx, legacyscheme.Scheme, obj, nil, allErrs, operation.Create,
    ) // No enforcement options passed
}
```

**4. `declarative_validation_test.go` (Equivalence Testing)**
Migrations require a specific test that proves the tags and the handwritten code produce the exact same errors.

```go
func TestDeclarativeValidation(t *testing.T) {
    apitesting.VerifyValidationEquivalence(t,
        &api.LegacyFeature{},
        func(obj runtime.Object) field.ErrorList {
            return strategy.Validate(context.TODO(), obj)
        },
    )
}
```

---

## 2. Faux Review: Catching Common Mistakes

Here are examples of what you should actively look for and push back on during a review.

### ❌ Mistake 1: Missing the Enforcement Flag (New APIs)

**The PR:** A developer adds `+k8s:required` to a new API field, writes tests using `.MarkNonShadowed()`, but forgets to update `strategy.go`.

**Why it's bad:** Without `rest.WithDeclarativeEnforcement()`, the `+k8s:required` tag is treated as an implicit shadow migration. It will generate a metric mismatch (because there's no handwritten code throwing the error), but it will *not* reject the invalid request. The API is effectively unvalidated.

**Your Review Comment:**
> *"I see you're adding DV tags for this new API, which is great! However, because this is a new API and there's no handwritten fallback, you need to ensure these tags are actually enforced. Please add `rest.WithDeclarativeEnforcement()` to the `ValidateDeclarativelyWithMigrationChecks` call in `strategy.go`."*

### ❌ Mistake 2: Missing Coverage Markers (Migrations)

**The PR:** A developer adds `+k8s:minimum=0` to `types.go` and wires up `strategy.go` correctly, but leaves `validation.go` exactly as it was.

**Why it's bad:** The equivalence test (`VerifyValidationEquivalence`) will fail. The DV engine will produce a `minimum` error, and the handwritten code will produce an `Invalid` error. Because the handwritten error lacks `.MarkCoveredByDeclarative()`, the testing framework flags this as a mismatch.

**Your Review Comment:**
> *"To complete this migration, you need to tell the framework that the handwritten error is now covered by the DV tag. Please update the `field.Invalid` call in `validation.go` to append `.MarkCoveredByDeclarative()`."*

### ❌ Mistake 3: Incomplete Version Migrations

**The PR:** A developer adds `+k8s:maxLength=64` to `staging/src/k8s.io/api/networking/v1beta1/types.go`, but forgets to add it to `.../v1/types.go` for the same field.

**Why it's bad:** DV runs on the *versioned* types. If a field exists in multiple versions, the tag must be applied to all of them to ensure consistent validation regardless of which API version the user interacts with.

**Your Review Comment:**
> *"It looks like this field also exists in the `v1` API. Please ensure the `+k8s:maxLength=64` tag is also added to the corresponding field in the `v1/types.go` file."*

### ❌ Mistake 4: Over-Testing Framework Logic

**The PR:** A developer adds `+k8s:format=k8s-short-name` and proceeds to write 50 test cases in `validation_test.go` checking every conceivable valid and invalid character combination for a DNS label.

**Why it's bad:** We trust the `validation-gen` framework to implement `k8s-short-name` correctly (it has its own exhaustive tests). Reviewing 50 redundant test cases wastes time.

**Your Review Comment:**
> *"Since we are using the standard `+k8s:format=k8s-short-name` tag, we don't need to exhaustively test the format itself here—the framework guarantees that. Let's reduce these test cases to just one or two basic valid/invalid examples to prove the tag is wired up to this specific field correctly."*

---

## 3. FAQ & Common Pitfalls

**Q: What if a field was renamed or moved between `v1beta1` and `v1`?**
A: DV validates the *versioned* type (e.g., `v1beta1`), but handwritten validation often validates the *internal* type (which usually matches `v1`). If the path changed (e.g., `spec.oldName` became `spec.nested.newName`), the migration equivalence check will fail due to a path mismatch.
*Solution:* The PR must introduce **Path Normalization Rules** in `strategy.go` (`rest.WithNormalizationRules`) to map the `v1beta1` path to the internal path before comparison.

**Q: Can DV handle cross-field validation (e.g., "Field A is required if Field B is true")?**
A: Currently, no. DV is focused on single-field validation constraints. Complex, cross-field logic must still be implemented in procedural Go code in `validation.go`.

**Q: The handwritten code checks 3 things, but DV short-circuits after the first failure. Is this okay?**
A: Yes, DV tags like `+k8s:optional`, `+k8s:required`, and `+k8s:immutable` are **short-circuiting**. If a field is missing, DV stops validating its minimums or formats. If handwritten code doesn't short-circuit, it will return more errors than DV.
*Solution:* The handwritten code often needs to be refactored to short-circuit similarly *before* or *during* the migration PR to achieve equivalence.

---

## 4. Advanced: The Validation Lifecycle (Under the Hood)

While you will mostly deal with standard tags (e.g., `+k8s:required`), you may occasionally see prefixed tags. The framework uses a lifecycle to safely introduce new rules.

*   **`+k8s:alpha(since:v1.XX)=...` (Shadow Mode):** Used when adding a *new* validation rule to an *existing* API field. The rule is executed, and mismatches are recorded in metrics, but the API request is **not rejected**. This allows us to gather data to ensure the new rule won't unexpectedly break existing clients.
    *   *Reviewer Action:* Verify the logic is sound. It's safe to merge as it's non-blocking.
*   **`+k8s:beta(since:v1.XX)=...` (Gated Enforcement):** The rule rejects invalid requests by default. However, cluster admins can disable it using the `DeclarativeValidationTakeover` feature gate if regressions occur.
    *   *Reviewer Action:* Ensure the rule has soaked in Alpha for at least one release before promotion to Beta.
*   **Standard Tags (No Prefix):** Permanently enforced. No global feature gate can turn them off.

### Summary Checklist for API Reviewers

1.  [ ] Are the `+k8s:` tags chosen appropriately from the official catalog?
2.  [ ] Are the tags applied consistently across all relevant API versions (v1, v1beta1, etc.)?
3.  [ ] **For New APIs:** Is `rest.WithDeclarativeEnforcement()` present in `strategy.go`? Are tests using `.MarkNonShadowed()`?
4.  [ ] **For Migrations:** Are handwritten errors marked with `.MarkCoveredByDeclarative()`? Is `VerifyValidationEquivalence` used in tests?
5.  [ ] Is the PR focused? (Ideally, N small atomic commits rather than one massive commit).
6.  [ ] Is cross-field logic appropriately left in handwritten code?

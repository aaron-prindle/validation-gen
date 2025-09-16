package deviceclass

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	genericapirequest "k8s.io/apiserver/pkg/endpoints/request"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	featuregatetesting "k8s.io/component-base/featuregate/testing"
	"k8s.io/kubernetes/pkg/apis/resource"
	_ "k8s.io/kubernetes/pkg/apis/resource/install" // Install the resource API group
	"k8s.io/kubernetes/pkg/features"
)

var apiVersions = []string{"v1", "v1beta1", "v1beta2"}

func TestDeclarativeValidate(t *testing.T) {
	for _, apiVersion := range apiVersions {
		t.Run(apiVersion, func(t *testing.T) {
			ctx := genericapirequest.WithRequestInfo(genericapirequest.NewDefaultContext(), &genericapirequest.RequestInfo{
				APIGroup:   "resource.k8s.io",
				APIVersion: apiVersion,
				Resource:   "deviceclasses",
			})

			strategy := Strategy

			testCases := map[string]struct {
				input        resource.DeviceClass
				expectedErrs field.ErrorList
			}{
				"valid basic class": {
					input: mkDeviceClass(),
				},
				// declarative validation is fully implemented for all fields
			}

			for k, tc := range testCases {
				t.Run(k, func(t *testing.T) {
					var declarativeTakeoverErrs field.ErrorList
					var imperativeErrs field.ErrorList

					for _, gateVal := range []bool{true, false} {
						featuregatetesting.SetFeatureGateDuringTest(t, utilfeature.DefaultFeatureGate, features.DeclarativeValidation, gateVal)
						featuregatetesting.SetFeatureGateDuringTest(t, utilfeature.DefaultFeatureGate, features.DeclarativeValidationTakeover, gateVal)

						errs := strategy.Validate(ctx, &tc.input)
						if gateVal {
							declarativeTakeoverErrs = errs
						} else {
							imperativeErrs = errs
						}

						errOutputMatcher := field.ErrorMatcher{}.ByType().ByField().ByOrigin()

						if len(tc.expectedErrs) > 0 {
							errOutputMatcher.Test(t, tc.expectedErrs, errs)
						} else if len(errs) != 0 {
							t.Errorf("expected no errors, but got: %v", errs)
						}
					}

					// Verify equivalence between imperative and declarative validation
					equivalenceMatcher := field.ErrorMatcher{}.ByType().ByField().ByOrigin()
					dedupedImperativeErrs := deduplicateErrors(imperativeErrs, equivalenceMatcher)
					equivalenceMatcher.Test(t, dedupedImperativeErrs, declarativeTakeoverErrs)
				})
			}
		})
	}
}

func TestDeclarativeValidateUpdate(t *testing.T) {
	for _, apiVersion := range apiVersions {
		t.Run(apiVersion, func(t *testing.T) {
			ctx := genericapirequest.WithRequestInfo(genericapirequest.NewDefaultContext(), &genericapirequest.RequestInfo{
				APIGroup:   "resource.k8s.io",
				APIVersion: apiVersion,
				Resource:   "deviceclasses",
			})

			strategy := Strategy

			testCases := map[string]struct {
				old          resource.DeviceClass
				update       resource.DeviceClass
				expectedErrs field.ErrorList
			}{
				"no changes": {
					old:    mkDeviceClass(),
					update: mkDeviceClass(),
				},
				// TODO: Add more test cases
			}

			for k, tc := range testCases {
				t.Run(k, func(t *testing.T) {
					// Set ResourceVersion for update validation
					tc.old.ResourceVersion = "1"
					tc.update.ResourceVersion = "1"

					var declarativeTakeoverErrs field.ErrorList
					var imperativeErrs field.ErrorList

					for _, gateVal := range []bool{true, false} {
						featuregatetesting.SetFeatureGateDuringTest(t, utilfeature.DefaultFeatureGate, features.DeclarativeValidation, gateVal)
						featuregatetesting.SetFeatureGateDuringTest(t, utilfeature.DefaultFeatureGate, features.DeclarativeValidationTakeover, gateVal)

						errs := strategy.ValidateUpdate(ctx, &tc.update, &tc.old)
						if gateVal {
							declarativeTakeoverErrs = errs
						} else {
							imperativeErrs = errs
						}

						errOutputMatcher := field.ErrorMatcher{}.ByType().ByField().ByOrigin()

						if len(tc.expectedErrs) > 0 {
							errOutputMatcher.Test(t, tc.expectedErrs, errs)
						} else if len(errs) != 0 {
							t.Errorf("expected no errors, but got: %v", errs)
						}
					}

					// Verify equivalence between imperative and declarative validation
					equivalenceMatcher := field.ErrorMatcher{}.ByType().ByField().ByOrigin()
					dedupedImperativeErrs := deduplicateErrors(imperativeErrs, equivalenceMatcher)
					equivalenceMatcher.Test(t, dedupedImperativeErrs, declarativeTakeoverErrs)
				})
			}
		})
	}
}

// Helper function to create a DeviceClass with default values and optional mutators
func mkDeviceClass(mutators ...func(*resource.DeviceClass)) resource.DeviceClass {
	dc := resource.DeviceClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-class",
		},
		Spec: resource.DeviceClassSpec{
			Selectors: []resource.DeviceSelector{
				{
					CEL: &resource.CELDeviceSelector{
						Expression: "device.driver == \"test.driver.io\"",
					},
				},
			},
			Config: []resource.DeviceClassConfiguration{
				{
					DeviceConfiguration: resource.DeviceConfiguration{
						Opaque: &resource.OpaqueDeviceConfiguration{
							Driver: "test.driver.io",
							Parameters: runtime.RawExtension{
								Raw: []byte(`{"key":"value"}`),
							},
						},
					},
				},
			},
		},
	}
	for _, mutate := range mutators {
		mutate(&dc)
	}
	return dc
}

func deduplicateErrors(errs field.ErrorList, matcher field.ErrorMatcher) field.ErrorList {
	dedupedErrs := field.ErrorList{}
	for _, err := range errs {
		found := false
		for _, existingErr := range dedupedErrs {
			if matcher.Matches(existingErr, err) {
				found = true
				break
			}
		}
		if !found {
			dedupedErrs = append(dedupedErrs, err)
		}
	}
	return dedupedErrs
}

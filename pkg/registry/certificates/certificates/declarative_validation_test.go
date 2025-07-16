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

package certificates

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
	genericapirequest "k8s.io/apiserver/pkg/endpoints/request"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	featuregatetesting "k8s.io/component-base/featuregate/testing"
	apitesting "k8s.io/kubernetes/pkg/api/testing"
	api "k8s.io/kubernetes/pkg/apis/certificates"
	"k8s.io/kubernetes/pkg/apis/core"
	"k8s.io/kubernetes/pkg/features"
	"k8s.io/utils/ptr"
)

var apiVersions = []string{"v1", "v1beta1"}

func TestDeclarativeValidateForDeclarative(t *testing.T) {
	for _, apiVersion := range apiVersions {
		testDeclarativeValidateForDeclarative(t, apiVersion)
	}
}

func testDeclarativeValidateForDeclarative(t *testing.T, apiVersion string) {
	ctx := genericapirequest.WithRequestInfo(genericapirequest.NewDefaultContext(), &genericapirequest.RequestInfo{
		APIGroup:   "certificates.k8s.io",
		APIVersion: apiVersion,
	})
	testCases := map[string]struct {
		input        api.CertificateSigningRequest
		expectedErrs field.ErrorList
	}{
		"no conditions - valid": {
			input: makeValidCSR(),
		},
		"single approved condition - valid": {
			input: makeValidCSR(withApprovedCondition()),
		},
		"single denied condition - valid": {
			input: makeValidCSR(withDeniedCondition()),
		},
		"single failed condition - valid": {
			input: makeValidCSR(withFailedCondition()),
		},
		"approved and failed conditions - valid": {
			input: makeValidCSR(withApprovedCondition(), withFailedCondition()),
		},
		"denied and failed conditions - valid": {
			input: makeValidCSR(withDeniedCondition(), withFailedCondition()),
		},
		"both approved and denied conditions - invalid": {
			input: makeValidCSR(withApprovedCondition(), withDeniedCondition()),
			expectedErrs: field.ErrorList{
				field.Invalid(
					field.NewPath("status", "conditions"),
					nil,
					"",
				).WithOrigin("zeroOrOneOf"),
			},
		},
		"denied then approved conditions - invalid": {
			input: makeValidCSR(withDeniedCondition(), withApprovedCondition()),
			expectedErrs: field.ErrorList{
				field.Invalid(
					field.NewPath("status", "conditions"),
					nil,
					"",
				).WithOrigin("zeroOrOneOf"),
			},
		},
	}
	for k, tc := range testCases {
		t.Run(k, func(t *testing.T) {
			var declarativeTakeoverErrs field.ErrorList
			var imperativeErrs field.ErrorList
			for _, gateVal := range []bool{true, false} {
				// We only need to test both gate enabled and disabled together, because
				// 1) the DeclarativeValidationTakeover won't take effect if DeclarativeValidation is disabled.
				// 2) the validation output, when only DeclarativeValidation is enabled, is the same as when both gates are disabled.
				featuregatetesting.SetFeatureGateDuringTest(t, utilfeature.DefaultFeatureGate, features.DeclarativeValidation, gateVal)
				featuregatetesting.SetFeatureGateDuringTest(t, utilfeature.DefaultFeatureGate, features.DeclarativeValidationTakeover, gateVal)

				errs := Strategy.Validate(ctx, &tc.input)
				if gateVal {
					declarativeTakeoverErrs = errs
				} else {
					imperativeErrs = errs
				}
				// The errOutputMatcher is used to verify the output matches the expected errors in test cases.
				errOutputMatcher := field.ErrorMatcher{}.ByType().ByField().ByOrigin()
				if len(tc.expectedErrs) > 0 {
					errOutputMatcher.Test(t, tc.expectedErrs, errs)
				} else if len(errs) != 0 {
					t.Errorf("expected no errors, but got: %v", errs)
				}
			}
			// The equivalenceMatcher is used to verify the output errors from hand-written imperative validation
			// are equivalent to the output errors when DeclarativeValidationTakeover is enabled.
			equivalenceMatcher := field.ErrorMatcher{}.ByType().ByField().ByOrigin()
			equivalenceMatcher.Test(t, imperativeErrs, declarativeTakeoverErrs)

			apitesting.VerifyVersionedValidationEquivalence(t, &tc.input, nil)
		})
	}
}

func TestValidateUpdateForDeclarative(t *testing.T) {
	for _, apiVersion := range apiVersions {
		testValidateUpdateForDeclarative(t, apiVersion)
	}
}

func testValidateUpdateForDeclarative(t *testing.T, apiVersion string) {
	ctx := genericapirequest.WithRequestInfo(genericapirequest.NewDefaultContext(), &genericapirequest.RequestInfo{
		APIGroup:   "certificates.k8s.io",
		APIVersion: apiVersion,
	})

	testCases := map[string]struct {
		old          api.CertificateSigningRequest
		update       api.CertificateSigningRequest
		expectedErrs field.ErrorList
	}{
		"no change in conditions - valid": {
			old:    makeValidCSR(withApprovedCondition()),
			update: makeValidCSR(withApprovedCondition()),
		},
		"ratcheting: approved+denied conditions unchanged - valid": {
			old:    makeValidCSR(withApprovedCondition(), withDeniedCondition()),
			update: makeValidCSR(withApprovedCondition(), withDeniedCondition()),
		},
		"ratcheting: approved+denied conditions, change spec - valid": {
			old: makeValidCSR(
				withApprovedCondition(),
				withDeniedCondition(),
				func(csr *api.CertificateSigningRequest) {
					csr.Spec.ExpirationSeconds = ptr.To(int32(3600))
				},
			),
			update: makeValidCSR(
				withApprovedCondition(),
				withDeniedCondition(),
				func(csr *api.CertificateSigningRequest) {
					csr.Spec.ExpirationSeconds = ptr.To(int32(7200))
				},
			),
		},
		"ratcheting: approved+denied conditions, add failed condition - valid": {
			old:    makeValidCSR(withApprovedCondition(), withDeniedCondition()),
			update: makeValidCSR(withApprovedCondition(), withDeniedCondition(), withFailedCondition()),
		},
	}
	for k, tc := range testCases {
		t.Run(k, func(t *testing.T) {
			tc.old.ResourceVersion = "1"
			tc.update.ResourceVersion = "1"
			var declarativeTakeoverErrs field.ErrorList
			var imperativeErrs field.ErrorList
			for _, gateVal := range []bool{true, false} {
				// We only need to test both gate enabled and disabled together, because
				// 1) the DeclarativeValidationTakeover won't take effect if DeclarativeValidation is disabled.
				// 2) the validation output, when only DeclarativeValidation is enabled, is the same as when both gates are disabled.
				featuregatetesting.SetFeatureGateDuringTest(t, utilfeature.DefaultFeatureGate, features.DeclarativeValidation, gateVal)
				featuregatetesting.SetFeatureGateDuringTest(t, utilfeature.DefaultFeatureGate, features.DeclarativeValidationTakeover, gateVal)
				errs := Strategy.ValidateUpdate(ctx, &tc.update, &tc.old)
				if gateVal {
					declarativeTakeoverErrs = errs
				} else {
					imperativeErrs = errs
				}
				// The errOutputMatcher is used to verify the output matches the expected errors in test cases.
				errOutputMatcher := field.ErrorMatcher{}.ByType().ByField().ByOrigin()

				if len(tc.expectedErrs) > 0 {
					errOutputMatcher.Test(t, tc.expectedErrs, errs)
				} else if len(errs) != 0 {
					t.Errorf("expected no errors, but got: %v", errs)
				}
			}
			// The equivalenceMatcher is used to verify the output errors from hand-written imperative validation
			// are equivalent to the output errors when DeclarativeValidationTakeover is enabled.
			equivalenceMatcher := field.ErrorMatcher{}.ByType().ByField().ByOrigin()
			// TODO: remove this once ErrorMatcher has been extended to handle this form of deduplication.
			dedupedImperativeErrs := field.ErrorList{}
			for _, err := range imperativeErrs {
				found := false
				for _, existingErr := range dedupedImperativeErrs {
					if equivalenceMatcher.Matches(existingErr, err) {
						found = true
						break
					}
				}
				if !found {
					dedupedImperativeErrs = append(dedupedImperativeErrs, err)
				}
			}
			equivalenceMatcher.Test(t, dedupedImperativeErrs, declarativeTakeoverErrs)

			apitesting.VerifyVersionedValidationEquivalence(t, &tc.update, &tc.old)
		})
	}
}

func TestValidateApprovalUpdateForDeclarative(t *testing.T) {
	for _, apiVersion := range apiVersions {
		testValidateApprovalUpdateForDeclarative(t, apiVersion)
	}
}

func testValidateApprovalUpdateForDeclarative(t *testing.T, apiVersion string) {
	ctx := genericapirequest.WithRequestInfo(genericapirequest.NewDefaultContext(), &genericapirequest.RequestInfo{
		APIGroup:          "certificates.k8s.io",
		APIVersion:        apiVersion,
		IsResourceRequest: true,
		Subresource:       "approval",
	})

	testCases := map[string]struct {
		old          api.CertificateSigningRequest
		update       api.CertificateSigningRequest
		expectedErrs field.ErrorList
	}{
		"no change in conditions - valid": {
			old:    makeValidCSR(withApprovedCondition()),
			update: makeValidCSR(withApprovedCondition()),
		},
		"add approved condition - valid from zeroOrOneOf perspective": {
			old:    makeValidCSR(),
			update: makeValidCSR(withApprovedCondition()),
		},
		"add approved+denied conditions - invalid": {
			old:    makeValidCSR(),
			update: makeValidCSR(withApprovedCondition(), withDeniedCondition()),
			expectedErrs: field.ErrorList{
				field.Invalid(
					field.NewPath("status", "conditions"),
					nil,
					"",
				).WithOrigin("zeroOrOneOf"),
			},
		},
		"ratcheting: approved+denied conditions unchanged - valid": {
			old:    makeValidCSR(withApprovedCondition(), withDeniedCondition()),
			update: makeValidCSR(withApprovedCondition(), withDeniedCondition()),
		},
		"ratcheting: approved+denied conditions, change spec - valid": {
			old: makeValidCSR(
				withApprovedCondition(),
				withDeniedCondition(),
				func(csr *api.CertificateSigningRequest) {
					csr.Spec.ExpirationSeconds = ptr.To(int32(3600))
				},
			),
			update: makeValidCSR(
				withApprovedCondition(),
				withDeniedCondition(),
				func(csr *api.CertificateSigningRequest) {
					csr.Spec.ExpirationSeconds = ptr.To(int32(7200))
				},
			),
		},
		"ratcheting: approved+denied conditions, add failed condition - valid": {
			old:    makeValidCSR(withApprovedCondition(), withDeniedCondition()),
			update: makeValidCSR(withApprovedCondition(), withDeniedCondition(), withFailedCondition()),
		},
		"ratcheting: approved+denied conditions, modify condition reason - valid": {
			old: makeValidCSR(
				func(csr *api.CertificateSigningRequest) {
					csr.Status.Conditions = []api.CertificateSigningRequestCondition{
						{Type: api.CertificateApproved, Status: core.ConditionTrue, Reason: "OldReason"},
						{Type: api.CertificateDenied, Status: core.ConditionTrue, Reason: "OldReason"},
					}
				},
			),
			update: makeValidCSR(
				func(csr *api.CertificateSigningRequest) {
					csr.Status.Conditions = []api.CertificateSigningRequestCondition{
						{Type: api.CertificateApproved, Status: core.ConditionTrue, Reason: "NewReason"},
						{Type: api.CertificateDenied, Status: core.ConditionTrue, Reason: "NewReason"},
					}
				},
			),
		},
	}
	for k, tc := range testCases {
		t.Run(k, func(t *testing.T) {
			tc.old.ResourceVersion = "1"
			tc.update.ResourceVersion = "1"
			var declarativeTakeoverErrs field.ErrorList
			var imperativeErrs field.ErrorList
			for _, gateVal := range []bool{true, false} {
				// We only need to test both gate enabled and disabled together, because
				// 1) the DeclarativeValidationTakeover won't take effect if DeclarativeValidation is disabled.
				// 2) the validation output, when only DeclarativeValidation is enabled, is the same as when both gates are disabled.
				featuregatetesting.SetFeatureGateDuringTest(t, utilfeature.DefaultFeatureGate, features.DeclarativeValidation, gateVal)
				featuregatetesting.SetFeatureGateDuringTest(t, utilfeature.DefaultFeatureGate, features.DeclarativeValidationTakeover, gateVal)
				errs := ApprovalStrategy.ValidateUpdate(ctx, &tc.update, &tc.old)
				if gateVal {
					declarativeTakeoverErrs = errs
				} else {
					imperativeErrs = errs
				}
				// The errOutputMatcher is used to verify the output matches the expected errors in test cases.
				errOutputMatcher := field.ErrorMatcher{}.ByType().ByField().ByOrigin()

				if len(tc.expectedErrs) > 0 {
					errOutputMatcher.Test(t, tc.expectedErrs, errs)
				} else if len(errs) != 0 {
					t.Errorf("expected no errors, but got: %v", errs)
				}
			}
			// The equivalenceMatcher is used to verify the output errors from hand-written imperative validation
			// are equivalent to the output errors when DeclarativeValidationTakeover is enabled.
			equivalenceMatcher := field.ErrorMatcher{}.ByType().ByField().ByOrigin()
			// TODO: remove this once ErrorMatcher has been extended to handle this form of deduplication.
			dedupedImperativeErrs := field.ErrorList{}
			for _, err := range imperativeErrs {
				found := false
				for _, existingErr := range dedupedImperativeErrs {
					if equivalenceMatcher.Matches(existingErr, err) {
						found = true
						break
					}
				}
				if !found {
					dedupedImperativeErrs = append(dedupedImperativeErrs, err)
				}
			}
			equivalenceMatcher.Test(t, dedupedImperativeErrs, declarativeTakeoverErrs)

			apitesting.VerifyVersionedValidationEquivalence(t, &tc.update, &tc.old)
		})
	}
}

func makeValidCSR(mutators ...func(*api.CertificateSigningRequest)) api.CertificateSigningRequest {
	csr := api.CertificateSigningRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-csr",
		},
		Spec: api.CertificateSigningRequestSpec{
			// Use the existing, reliable helper from the other test file
			Request:    newCSRPEM(&testing.T{}),
			SignerName: "example.com/signer",
			Usages:     []api.KeyUsage{api.UsageDigitalSignature, api.UsageKeyEncipherment},
		},
	}
	for _, mutate := range mutators {
		mutate(&csr)
	}
	return csr
}

func newCSRPEM(t *testing.T) []byte {
	template := &x509.CertificateRequest{
		Subject: pkix.Name{
			Organization: []string{"testing-org"},
		},
	}

	_, key, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	csrDER, err := x509.CreateCertificateRequest(rand.Reader, template, key)
	if err != nil {
		t.Fatal(err)
	}

	csrPemBlock := &pem.Block{
		Type:  "CERTIFICATE REQUEST",
		Bytes: csrDER,
	}

	p := pem.EncodeToMemory(csrPemBlock)
	if p == nil {
		t.Fatal("invalid pem block")
	}

	return p
}

func withApprovedCondition() func(*api.CertificateSigningRequest) {
	return func(csr *api.CertificateSigningRequest) {
		csr.Status.Conditions = append(csr.Status.Conditions, api.CertificateSigningRequestCondition{
			Type:   api.CertificateApproved,
			Status: core.ConditionTrue,
		})
	}
}

func withDeniedCondition() func(*api.CertificateSigningRequest) {
	return func(csr *api.CertificateSigningRequest) {
		csr.Status.Conditions = append(csr.Status.Conditions, api.CertificateSigningRequestCondition{
			Type:   api.CertificateDenied,
			Status: core.ConditionTrue,
		})
	}
}

func withFailedCondition() func(*api.CertificateSigningRequest) {
	return func(csr *api.CertificateSigningRequest) {
		csr.Status.Conditions = append(csr.Status.Conditions, api.CertificateSigningRequestCondition{
			Type:   api.CertificateFailed,
			Status: core.ConditionTrue,
		})
	}
}

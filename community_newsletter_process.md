# Newsletter Process for Declarative Validation

## Purpose

The Declarative Validation Working Group (DV WG) sends a regular newsletter to Kubernetes API Reviewers and SIG Leads. The goal is to keep them informed about the progress of the Declarative Validation initiative, explain new features and how to review them, and solicit feedback on complex validation cases.

## Cadence

The newsletter should be sent at least once per release, ideally early in the release cycle to establish expectations for upcoming API reviews. A follow-up newsletter may be sent mid-release if there are significant updates or "Hard Cases" that require feedback.

## Audience

*   **Primary:** `kubernetes-api-reviewers@googlegroups.com`
*   **Secondary:** `kubernetes-sig-leads@googlegroups.com`
*   **Internal Relay:** Appropriate internal open-source facing teams at Google (e.g., via `aprindle@google.com`).

## Drafting Process

1.  **Drafting:** Create a draft document (Google Doc or markdown file) outlining the key topics for the upcoming release.
2.  **Review:** Circulate the draft among the DV WG members (Joe Betz, Lalit Chauhan, Yongrui Lin, Aaron Prindle, etc.) for feedback and approval.
3.  **Finalize:** Incorporate feedback and finalize the content.
4.  **Send:** Send the email to the target mailing lists.

## Newsletter Template (Draft for v1.36)

To: `kubernetes-api-reviewers@googlegroups.com`, `kubernetes-sig-leads@googlegroups.com`
From: `aprindle@google.com` (on behalf of the Declarative Validation WG)
Subject: Streamlining API Reviews: Reclaiming Time with Declarative Validation (v1.36)

---

**Streamlining API Reviews: Reclaiming Time with Declarative Validation**

Declarative Validation (DV) is the recommended way of authoring API validation for Kubernetes starting in v1.36.

The plan for v1.36 is to start a pilot program designed to ramp API Reviewers up on Declarative Validation generally as well as the new **Validation Lifecycle Mechanism** introduced in v1.36. We are also offering "shadowing" opportunities to learn from DV WG members and establishing communication channels to allow for back-and-forth communication.

**Evidence Of Success Using Declarative Validation**

*   Over 370 validations have been migrated to DV so far (excluding `+k8s:optional` tags).
*   In v1.35, ~30% of the Dynamic Resource Allocation (DRA) validation logic (“resource” group) was successfully migrated from hand-written code to declarative tags.
*   15 contributors have successfully landed DV migration changes.

**What is Declarative Validation and Why It Matters For Review**

Instead of going through blocks of `pkg/apis/.../validation.go` logic, cross-referencing with `types.go` files for errors/tag-consistency, and checking alternative `*/validation.go` files for validation consistency—validation is now consistent and self-documenting in `types.go`:

*Before: ReplicationControllerSpec handwritten validation method with 10+ lines of Go boilerplate.*

*After: Validation logic generated from declarative tags in types.go file*

```go
// ReplicationControllerSpec is the specification of a replication controller.
type ReplicationControllerSpec struct {
...
	// +k8s:optional
	// +default=1
	// +k8s:minimum=0
	Replicas *int32 `json:"replicas,omitempty" protobuf:"varint,1,opt,name=replicas"`

	// +k8s:optional
	// +default=0
	// +k8s:minimum=0
	MinReadySeconds int32 `json:"minReadySeconds,omitempty" protobuf:"varint,4,opt,name=minReadySeconds"`
```

**The Impact - Easier API Reviews:**

*   **Single Source of Truth:** Validation is co-located with the type definition; no more jumping between files and making sure logic is consistent across files.
*   **Validation Consistency:** Standardized rules are applied identically across all API versions and all APIs.
*   **Automated Catching Of Validation Errors In CI:** Linters (kube-api-linter w/ DV rules + validation-gen internal logic) catch common errors in CI before API Review.

*Future Additional Impact:* Documentation generation, CRD Support.

**What to Expect in v1.36: The Validation Lifecycle**

In the upcoming release, you will see an increase in PRs utilizing DV. In v1.36, we are introducing a data-driven **Validation Lifecycle Mechanism** utilizing `+k8s:alpha` and `+k8s:beta` tag prefixes. This replaces the previous implicit shadowing model and global feature gates, allowing for fine-grained control over when declarative rules are enforced.

For net-new APIs, you will see standard tags used immediately to enforce validation natively. For existing fields adopting new rules, you will see `+k8s:alpha` tags used to gather data before enforcement.

```go
type NewFeatureSpec struct {
    // This is Authoritative (Enforced immediately for new APIs)
    // +k8s:required
    // +k8s:minimum=1
    // +k8s:maximum=100
    BurstLimit int32 `json:"burstLimit"`

    // This is Shadowed (Gathers metrics but does not reject requests)
    // +k8s:alpha(since:v1.36)=+k8s:maxLength=512
    Reason string `json:"reason"`
}
```

**Your Role:** The DV WG wants to support API Reviewers to understand this new lifecycle and feel confident reviewing these tags. The DV WG has prepared an API Reviewer Guide to aid in this: `<REVIEWER_GUIDE TBD>`

The current plan for v1.36 is to partner with the Workload API to use declarative enforcement for all supported fields. We are actively looking for other new APIs with net-new fields and validation logic to participate. If you have any candidates, please reach out.

**DV Documentation - Where to Learn More**

The DV WG has a general set of documentation outlining Declarative Validation, the tag catalog, and user guides:

*   **Declarative Validation KEP [KEP-5073] (k/enhancements):** Specifically the new "Validation Lifecycle Mechanism and Updated Rollout Plan" section.
*   **Declarative Validation Documentation Page (k8s.io)**
*   **Declarative Validation Tag Catalog (k8s.io):** Tells what tags are “Stable”.
*   **API Changes Guide - DV Section (k/community)**

**Join the Pilot Program**

The Declarative Validation WG is launching a pilot program specifically for API reviewers around DV. The goal is to get API Reviewers onboarded onto reviewing DV PRs and to increase communication across the DV and API Reviewer members.

**1. The Shadowing Program**

For v1.36+, there will be a program to shadow Declarative Validation WG Members on relevant DV k/k PRs.

*   **How it works:** Sign up on the sign-up sheet here: [DV + API Reviewers Shadow Review Sign-up](#). You will be assigned a DV WG Member to shadow and be cc’d on appropriate DV PRs for shadowing.
*   **The Ask:** Walk through a "DV-Native" review with your assigned DV WG Member. Ask questions, challenge tag usage, and build muscle memory.

**2. Co-Design & Feedback**

Some validation logic is hard to capture in tags (e.g., cross-field validations, unions). The DV WG will bring design proposals for these "Hard Cases" to API Reviewers via this newsletter and by attending SIG Architecture meetings. We need your feedback to ensure these tags meet the high bar of Kubernetes API quality.

**Engagement Channels**

*   **Slack:** `#sig-api-machinery-dev-tools` – The channel where the DV WG discusses the project.
*   **SIG Architecture:** The DV WG will bring major design topics and status updates here for broad alignment.
*   **Declarative APIs Subproject Meeting:** Bi-weekly on Tuesdays @ 9:00am PST - Next session 1/27.

**Call to Action**

*   **Join the Channel:** `#sig-api-machinery-dev-tools`.
*   **Join the Shadowing Program:** Let us know if you're interested in shadowing a review.
*   **Read the Guides:** Check out the resources linked above to familiarize yourself with the new Validation Lifecycle.

Cheers,
The Declarative Validation Working Group

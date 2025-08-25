# **Go Test Auth Package**

This package provides test helpers for validating Google Cloud Platform (GCP) authentication and authorization within your Go integration tests. It ensures that tests that interact with real GCP services fail fast with clear, actionable error messages if the necessary credentials or permissions are not in place.

## **Features 🚀**

* **Fail-Fast Credential Checks**: Verifies that Application Default Credentials (ADC) are configured correctly.
* **Clear Error Messages**: Provides detailed, user-friendly error messages telling the developer exactly how to fix their authentication issues.
* **Permission Validation**: Checks for specific permissions required for advanced operations, like invoking Cloud Run services.
* **Automatic Test Skipping**: Skips tests gracefully if the GCP\_PROJECT\_ID environment variable isn't set, preventing failures in environments without GCP access.

## **Usage**

These helpers are designed to be called at the beginning of any test function that makes calls to live GCP services.

### **CheckGCPAuth**

Use this for tests that perform standard resource management (e.g., creating Pub/Sub topics, reading from GCS). It verifies that the user has authenticated with gcloud and that the credentials are valid.

**Example**:

import (  
"testing"  
"github.com/your/repo/auth" // Update with your import path  
)

func TestMyIntegrationWithGCP(t \*testing.T) {  
// This check will fail the test with a helpful message if ADC is not set up.  
// It will skip the test if GCP\_PROJECT\_ID is not set.  
projectID := auth.CheckGCPAuth(t)

    // ... rest of your test logic using projectID ...  
}

If the user has not run gcloud auth application-default login, the test will fail with a message like this:

\---------------------------------------------------------------------  
GCP AUTHENTICATION FAILED\!  
\---------------------------------------------------------------------  
Could not create a Google Cloud client. This is likely due to  
expired or missing Application Default Credentials (ADC).

To fix this, please run 'gcloud auth application-default login'.

Original Error: ...  
\---------------------------------------------------------------------

### **CheckGCPAdvancedAuth**

Use this for tests that need to perform actions requiring specific IAM roles, such as invoking a secure Cloud Run service. In addition to the basic ADC check, it also verifies that the credentials can be used to generate ID tokens.

**Example**:

import (  
"testing"  
"github.com/your/repo/auth" // Update with your import path  
)

func TestCloudRunInvocation(t \*testing.T) {  
// This checks for the 'Cloud Run Invoker' permission in addition to basic auth.  
// The 'true' argument tells it to log the principal being used.  
projectID := auth.CheckGCPAdvancedAuth(t, true)

    // ... your test logic to invoke a Cloud Run service ...  
}

If the user's credentials are valid but they lack the roles/run.invoker permission, the test will fail with a specific, actionable error message:

\---------------------------------------------------------------------  
GCP INVOCATION AUTHENTICATION FAILED\!  
\---------------------------------------------------------------------  
The test failed because your Application Default Credentials (ADC)  
are user credentials, which cannot be used by this client library to  
invoke secure Cloud Run services directly.

To fix this, the user running the test needs the permission to invoke  
Cloud Run services.

SOLUTION: Grant your user the 'Cloud Run Invoker' role on the project.  
1\. Find your user email by running:  
gcloud auth list \--filter=status:ACTIVE \--format="value(account)"  
2\. Grant the role by running (replace \[YOUR\_EMAIL\] and \[YOUR\_PROJECT\]):  
gcloud projects add-iam-policy-binding \[YOUR\_PROJECT\] \--member="user:\[YOUR\_EMAIL\]" \--role="roles/run.invoker"

After granting the permission, you may need to refresh your credentials:  
gcloud auth application-default login

Original Error: ...  
\---------------------------------------------------------------------  

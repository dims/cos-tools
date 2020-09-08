# COS Changelog Webapp
A web application that generates a changelog between 2 builds based on the commit difference between them.

## Setup
### App Engine
Create a new App Engine project to host this application. Detailed instructions are located [here](https://cloud.google.com/appengine/docs/standard/nodejs/building-app/creating-project).

### Secret Manager
This application queries secret manager for any information that should not be publicly accessible, such as client secrets and internal instance URLs. Ensure this service is enabled in Google Cloud, and that the App Engine service account has the `Secret Manager Secret Accessor` role in Google Cloud IAM.

## Configuration
`app.yaml` stores public environment variables used to run the service. For private environment variables, the variable name is stored in Secret Manager instead. This is indicated by the `_NAME` suffix for any environment variable.

For each secret name defined in `app.yaml`, a corresponding secret must be made in Google Secret Manager under the same variable name. See [here](https://cloud.google.com/secret-manager/docs/quickstart#secretmanager-quickstart-web) for more information on managing secrets. Secrets must be made for the Oauth client secret, session secret, internal repository names, and internal Gerrit/Git on Borg URLs.

## Deployment
Install [Cloud SDK](https://cloud.google.com/sdk/docs) and configure it to use the Google Cloud project you want to deploy to.

Clone the `cos/tools` repository and `cd` to the `src/cmd/changelog-webapp` directory. `app.yaml` should be in this directory.

Run `gcloud app deploy` to deploy the application. You can view the application with `gcloud app browse` after deployment is complete.
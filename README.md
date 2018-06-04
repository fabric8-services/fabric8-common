# fabric8-common
Common packages for the fabric8 services

This repository addresses issue #3553 https://github.com/openshiftio/openshift.io/issues/3553

Go code shared between fabric8 services can be added here, and then imported by the services.
This will add consistency, reliability, clarity, and (hopefully) reduce bugs and circular repo-to-repo dependencies.

Any code added to this repository should also have test code added, that passes "go test .".

Contents of this repository are anticipated to include:

_Configuration_

_Event bus_

_Logging_
- Common logging format
- Single initialization
- Error handling
- Common HTTP error response format
- Metrics

_Utility_
- Validation routines (application name, etc)
- HTTP/REST (closing result body, URL utils)

_Auth_
- Loading/parsing public key
- Service Account token management
- JWT token parsing (jwt_token to token_string and token_string to jwt_token)

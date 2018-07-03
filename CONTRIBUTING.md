This project contains common golang packages used by other projects.

Currently, this project doesn't have any standard build mechanism.

To contribute, follow below steps:

1. Clone the git repo
2. Run 'dep ensure -vendor-only' on root directory.

Running 'dep ensure -vendor-only' will create vendor directory with all dependency.

Now, you can build and test each package individually.  For ex, if you want to contribute to `metric` pakcage.

For build - go build ./metric/
For test - go test ./metric/

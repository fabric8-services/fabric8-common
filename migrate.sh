#!/bin/bash

# Output command before executing
# set -x

# Exit on error
set -e

# export the commits (as patches) from the source repo
echo "exporting content and history of $2 from $1" 
pushd $GOPATH/src/github.com/fabric8-services/$1 1>/dev/null
git log --pretty=email --patch-with-stat --reverse --full-index --binary -- $2 > /tmp/$2.patch
echo "(naively) converting goimports from \"github.com/fabric8-services/$1\" to \"github.com/fabric8-services/fabric8-common\"..."

# replace imports of root pkg
SED_REGEX="s/\"github.com\/fabric8-services\/$1\"$/\"github.com\/fabric8-services\/fabric8-common\"/g"
sed -i -e $SED_REGEX /tmp/$2.patch
# rename imports of sub pkg
SED_REGEX="s/\"github.com\/fabric8-services\/$1\/\([a-zA-Z0-9/]*\)\"$/\"github.com\/fabric8-services\/fabric8-common\/\1\"/g"
sed -i -e $SED_REGEX /tmp/$2.patch 
# check the changes
grep "github.com/fabric8-services" /tmp/$2.patch 

# import the commits into the target repo
popd 1>/dev/null
echo "importing $2 content and history into `pwd`"
git am /tmp/$2.patch 

# build to verify
go build $2

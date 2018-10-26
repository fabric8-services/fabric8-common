#!/bin/bash

function git_configure_and_clone() {
    git config --global user.name "FABRIC8 CD autobot"
    git config --global user.email fabric8cd@gmail.com

    set +x
    echo git clone https://XXXX@github.com/${GHORG}/${GHREPO}.git --depth=1 /tmp/${GHREPO}
    git clone https://$(echo ${FABRIC8_HUB_TOKEN}|base64 --decode)@github.com/${GHORG}/${GHREPO}.git --depth=1 /tmp/${GHREPO}
    set -x
}

function generate_client_and_create_pr() {
    make docker-generate-client

    local newVersion=${LATEST_COMMIT}
    local message="chore: update client version to ${newVersion}"
    local body=$(pr_body)
    local short_head=$(git rev-parse --short HEAD)
    local branch="client_update_${short_head}_$(date +%s)"

    cd /tmp/${GHREPO}
    git checkout -b ${branch}
    cd -
    for i in $(find tool cluster -name "*.go"); do
        sed -i 's:"github.com/'${GHORG}'/'${SERVICE_NAME}'/'${PKG_NAME}'":"github.com/'${GHORG}'/'${GHREPO}'/'${PKG_NAME}'":' "$i";
        sed -i 's:src/github.com/'${GHORG}'/'${SERVICE_NAME}':src/github.com/'${GHORG}'/'${GHREPO}':' "$i";
        sed -i 's:"github.com/'${GHORG}'/'${SERVICE_NAME}'/'${TOOL_DIR}'/cli":"github.com/'${GHORG}'/'${GHREPO}'/'${TOOL_DIR}'/cli":' "$i";
    done
    rm -rf /tmp/${GHREPO}/cluster /tmp/${GHREPO}/tool
    cp -r cluster tool /tmp/${GHREPO}
    git rev-parse HEAD > /tmp/${GHREPO}/source_commit.txt
    cd /tmp/${GHREPO}

    git commit cluster tool source_commit.txt -m "${message}"
    git push -u origin ${branch}

    set +x
    curl -s -X POST -L -H "Authorization: token $(echo ${FABRIC8_HUB_TOKEN}|base64 --decode)" \
         -d "{\"title\": \"${message}\", \"body\": \"$(echo $body)\", \"base\":\"master\", \"head\":\"${branch}\"}" \
         https://api.github.com/repos/${GHORG}/${GHREPO}/pulls
    set -x
}

function pr_body() {
    local description=$(cat <<EOF
            **About**<br><br>
            This description was generated using following command:<br><br>
            \`\`\`

            `echo GHORG=${GHORG} GHREPO=${GHREPO} LAST_USED_COMMIT=${LAST_USED_COMMIT} LATEST_COMMIT=${LATEST_COMMIT} \
             git log --pretty="%n**Commit:** https://github.com/${GHORG}/${GHREPO}/commit/%H%n**Author:** %an (%ae)%n**Date:** %aI%n%n" --reverse ${LAST_USED_COMMIT}..${LATEST_COMMIT} design
           `

            \`\`\`
            <br><br>
            **Commits with change in Design Package**<br><br>
EOF
)

    local commits=$(git log --pretty="**Commit:** https://github.com/${GHORG}/${GHREPO}/commit/%H<br>**Author:** %an (%ae)<br>**Date:** %ai<br><br>" --reverse ${LAST_USED_COMMIT}..${LATEST_COMMIT} design)

    echo $description$commits
}

function generate_client_setup() {
    SERVICE_NAME=${1}
    PKG_NAME=${2}               # Name of generated client Go package used in `goagen client --pkg PKG_NAME`
    TOOL_DIR=${3:-tool}         # Name of generated tool directory used in `goagen client --tooldir TOOL_DIR`
    GHORG=${4:-fabric8-services}
    GHREPO=${5:-${SERVICE_NAME}-client}
    LAST_USED_COMMIT=$(curl -s https://raw.githubusercontent.com/${GHORG}/${GHREPO}/master/source_commit.txt)
    LATEST_COMMIT=$(git rev-parse HEAD)
    if [[ $(git diff --reverse $LAST_USED_COMMIT..$LATEST_COMMIT design) ]]; then
        echo "generating new client."
        git_configure_and_clone
        generate_client_and_create_pr
    else
        echo "no change in design package."
    fi
}

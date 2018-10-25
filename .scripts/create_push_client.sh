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
    local branch="client_update_${short_head}"

    cd /tmp/${GHREPO}
    git checkout -b ${branch}
    cd -
    cp -r cluster tool /tmp/${GHREPO}
    git rev-parse --short HEAD > /tmp/${GHREPO}/source_commit.txt
    cd /tmp/${GHREPO}

    git commit cluster tool source_commit.txt -m "${message}"
    git push -u origin ${branch}
    rm -rf /tmp/${GHREPO}

    set +x
    curl -s -X POST -L -H "Authorization: token $(echo ${FABRIC8_HUB_TOKEN}|base64 --decode)" \
         -d "{\"title\": \"${message}\", \"body\": \"${body}\", \"base\":\"master\", \"head\":\"${branch}\"}" \
         https://api.github.com/repos/${GHORG}/${GHREPO}/pulls
    set -x
}

function pr_body() {
    local body=$(cat <<EOF
    # About
    This description was generated using following command:
    \`\`\`sh

    `echo GHORG=${GHORG} GHREPO=${GHREPO} LAST_USED_COMMIT=${LAST_USED_COMMIT} LATEST_COMMIT=${LATEST_COMMIT} \
    git log --pretty="%n**Commit:** https://github.com/${GHORG}/${GHREPO}/commit/%H%n**Author:** %an (%ae)%n**Date:** %aI%n%n%s%n%n%b%n%n----%n" \
            --reverse ${LAST_USED_COMMIT}..${LATEST_COMMIT} \
            | sed -E "s/([\s|\(| ])#([0-9]+)/\1${GHORG}\/${GHREPO}#\2/g"`

    \`\`\`

    # Changes
EOF
    git log \
      --pretty="%n**Commit:** https://github.com/${GHORG}/${GHREPO}/commit/%H%n**Author:** %an (%ae)%n**Date:** %aI%n%n%s%n%n%b%n%n----%n" \
      --reverse ${LAST_USED_COMMIT}..${LATEST_COMMIT} \
      | sed -E "s/([\s|\(| ])#([0-9]+)/\1${GHORG}\/${GHREPO}#\2/g"
)

    echo $body
}

function generate_client_setup() {
    SERVICE_NAME=${PWD##*/}
    GHORG=${1:-fabric8-services}
    GHREPO=${2:-${SERVICE_NAME}-client}
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

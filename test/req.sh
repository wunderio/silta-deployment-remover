#!/bin/bash

address=$1
file=$2
repository=$3
branchname=$4
event=$5

# Check if all flags are set
if [ -z "${address}" ] || [ -z "${file}" ] || [ -z "${repository}" ] || [ -z "${branchname}" ] || [ -z "${event}" ]
then
    echo "usage: req.sh <address> <file> <repository> <branchname> <event>"
    exit 1
fi

if [ ! -f ${file} ]
then
    echo "payload file does not exist"
    exit 1
fi

# Create temp file with yaml extension
tmpfile=$(mktemp --suffix=.yaml)

# Copy file and strip newlines as it messes up checkum calculation
tr -d "\n" < ${file} > ${tmpfile}

cat ${tmpfile}
echo "-------"

# Replace placeholders with actual values
sed -i "s|{{REPOSITORY}}|${repository}|g" ${tmpfile}
sed -i "s|{{BRANCHNAME}}|${branchname}|g" ${tmpfile}

if [[ "$file" == *"github"* ]]; then
    sig=$(cat "${tmpfile}" | openssl dgst -sha1 -hmac "${WEBHOOKS_SECRET}" | awk '{print "X-Hub-Signature: sha1="$2}')
    sig256=$(cat "${tmpfile}" | openssl dgst -sha256 -hmac "${WEBHOOKS_SECRET}" | awk '{print "X-Hub-Signature-256: sha256="$2}')
    # sig=$(echo -n "${payload}" | openssl dgst -sha1 -hmac "${WEBHOOKS_SECRET}" | awk '{print "X-Hub-Signature: sha1="$2}')

    curl -v -d @${tmpfile} -H "Content-Type: application/json" -H "X-GitHub-Event: ${event}" -H "${sig}" -H "${sig256}" "${address}"
fi

if [[ "$file" == *"azure"* ]]; then
    activity_id="aaaaaaaa-bbbb-cccc-dddd-123456789012",
    basic_auth="-u :${WEBHOOKS_SECRET}" # include -u in case secret is ommited
    curl -v -d @${tmpfile} ${basic_auth} -H "X-Vss-Activityid: ${activity_id}" -H "Content-Type: application/json" "${address}"
fi

rm ${tmpfile}

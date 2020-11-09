#!/usr/bin/env python

import os, sys, requests, json, yaml, re, time
from kubernetes import client, config
from dotenv import load_dotenv

def conf_read():
    global github_api_token

    load_dotenv()
    github_api_token = os.environ.get('GITHUB_API_TOKEN')

def conf_parse(config, attribute):
    if config and attribute in config:
        return config[attribute]
    else:
        return None

# Github API request abstraction
def github_api_request(resource):

    url = "https://api.github.com/{}?per_page=100".format(resource)
    res=requests.get(url,headers={"Authorization": "token {}".format(github_api_token)})
    response_json=res.json()
    while 'next' in res.links.keys():
        res=requests.get(res.links['next']['url'],headers={"Authorization": "token {}".format(github_api_token)})
        response_json.extend(res.json())

    return response_json

def get_namespaces():
    
    namespaces = kubeAPI.list_namespace()
    return namespaces.items

def get_branches(repository):
    return github_api_request('repos/{}/{}/branches'.format('wunderio', repository))

##

conf_read()

if not github_api_token:
    print ("No GITHUB_API_TOKEN defined!")
    sys.exit()

config.load_kube_config()
kubeAPI = client.CoreV1Api()

removal_commands = []
    
for namespace in get_namespaces():

    namespace_name = namespace.metadata.name
    print("* Processing {}".format(namespace_name))

    # Get all deployed branches
    helm_releases = {}
    try:
        api_response = kubeAPI.list_namespaced_config_map(namespace_name, limit=0, timeout_seconds=5, watch=False)
        for cm in api_response.items:
            # Get release name from branch name
            if hasattr(cm, 'data') and ('branchName' in cm.data):
                helm_releases[cm.metadata.labels['release']] = cm.data['branchName']

    except Exception as e:
        print("!! Error: {}".format(e))

    # Get list of branches from repository
    repository_branches = []
    for branch in get_branches(namespace_name):
        if 'name' in branch:
            repository_branches.append(branch['name'])

    if (len(repository_branches) > 0):
        print("  Finding releases that do not have branch")

        # Find the releases that do not have branch 
        for releasename in helm_releases:
            branchname = helm_releases[releasename]
            if branchname not in repository_branches:
                print('    Will remove release {} because branch {} does not exist.'.format(releasename, branchname))
                removal_commands.append('helm delete -n {} {}'.format(namespace_name, releasename))
    else:
        print "  ! No branches found"
    
    # newline
    print "" 

print('### Review and execute these commands manually')

# Print commands
for command in removal_commands:
    print(command)

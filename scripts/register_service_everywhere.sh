#!/bin/bash

# USAGE: ./scripts/register_service_everywhere.sh

# REQUIREMENTS: jq & cf CLIs

set -e # fail fast

current_org=$(cat ~/.cf/config.json | jq -r ".OrganizationFields.Name")
current_space=$(cat ~/.cf/config.json | jq -r ".OrganizationFields.Name")

echo "Run the following command to return to current org/space:"
echo "cf target -o \"${current_org}\" -s \"${current_space}\""
echo

orgs_next_page_url="/v2/organizations"
while [[ "${orgs_next_page_url}" != "null" ]]; do
  orgs_page=$(cf curl $orgs_next_page_url)
  orgs_next_page_url=$(echo $orgs_page | jq -r .next_url)
  orgs_page_count=$(echo $orgs_page | jq -r ".resources | length")
  for (( org = 0; org < ${orgs_page_count}; org++ )); do
    org_guid=$(echo $orgs_page | jq -r ".resources[$org].metadata.guid")
    org_name=$(echo $orgs_page | jq -r ".resources[$org].entity.name")
    spaces_next_page_url=$(echo $orgs_page | jq -r ".resources[$org].entity.spaces_url")
    echo ${org_name} ...
    
    while [[ "${spaces_next_page_url}" != "null" ]]; do
      spaces_page=$(cf curl $spaces_next_page_url)
      spaces_next_page_url=$(echo $spaces_page | jq -r .next_url)
      spaces_page_count=$(echo $spaces_page | jq -r ".resources | length")
      for (( space = 0; space < ${spaces_page_count}; space++ )); do
        echo ${org_name} "/" $(echo $spaces_page | jq -r ".resources[$space].entity.name")
      done
    done
  done
done

#!/bin/bash

# USAGE: ./scripts/register_service_everywhere.sh dingo-s3 username password baseurl

# REQUIREMENTS: jq & cf CLIs

base_broker_name=$1; shift
base_broker_username=$1; shift
base_broker_password=$1; shift
base_broker_url=$1; shift
if [[ "${base_broker_url}X" == "X" ]]; then
  echo "USAGE: ./scripts/register_service_everywhere.sh broker-name username password baseurl"
  exit 1
fi

set -e # fail fast

current_org=$(cat ~/.cf/config.json | jq -r ".OrganizationFields.Name")
current_space=$(cat ~/.cf/config.json | jq -r ".SpaceFields.Name")

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
        space_guid=$(echo $spaces_page | jq -r ".resources[$space].metadata.guid")
        space_name=$(echo $spaces_page | jq -r ".resources[$space].entity.name")
        echo ${org_name} "/" ${space_name}

        space_broker_name="${base_broker_name}-${org_name}-${space_name}"
        space_broker_url="${base_broker_url}/${org_name}-${space_name}"
        space_brokers=$(cf curl "/v2/service_brokers?q=space_guid:${space_guid}")
        space_brokers_count=$(echo $space_brokers | jq -r ".resources | length")
        echo ${space_broker_name} ${space_broker_url} "-" ${space_brokers_count}
        space_broker_guid_found=

        for (( bkr = 0; bkr < ${space_brokers_count}; bkr++ )); do
          bkr_guid=$(echo $space_brokers | jq -r ".resources[$bkr].metadata.guid")
          bkr_name=$(echo $space_brokers | jq -r ".resources[$bkr].entity.name")
          bkr_url=$(echo $space_brokers | jq -r ".resources[$bkr].entity.broker_url")
          if [[ "${space_broker_name}" == "${bkr_name}" || "${space_broker_url}" == "${bkr_url}" ]]; then
            echo "Broker already exists" $bkr_name " - updating..."
            space_broker_guid_found=${bkr_guid}
          fi
        done
        if [[ "${space_broker_guid_found}X" == "X" ]]; then
          echo "Creating broker..."
          cf curl /v2/service_brokers -X POST -d "{\"space_guid\": \"${space_guid}\", \"name\": \"${space_broker_name}\", \"broker_url\": \"${space_broker_url}\", \"auth_username\": \"${base_broker_username}\", \"auth_password\": \"${base_broker_password}\"}" -H "Content-Type: application/x-www-form-urlencoded"
        else
          cf curl /v2/service_brokers/${space_broker_guid_found} -X PUT -d "{\"space_guid\": \"${space_guid}\", \"name\": \"${space_broker_name}\", \"broker_url\": \"${space_broker_url}\", \"auth_username\": \"${base_broker_username}\", \"auth_password\": \"${base_broker_password}\"}" -H "Content-Type: application/x-www-form-urlencoded"
        fi
      done
    done
  done
done

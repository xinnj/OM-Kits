#! /bin/bash
set -euao pipefail

not_ready_pod_list() {
  kubectl get pods -A -o=custom-columns=':.metadata.namespace,:.metadata.name,:.status.containerStatuses[*].ready,:.status.phase,:.status.containerStatuses[*].restartCount,:.metadata.ownerReferences[*].kind' | grep -v -E "^(\S+\s+)(\S+\s+)(\S+\s+)(\S+\s+)(\S+\s+)Job" | grep -E "^(\S+\s+)(\S+\s+)\S*?false\S*?\s+" | awk '{print $1,$2}' || true
}

display_not_ready_pod() {
  kubectl get pods -A -o=custom-columns='NAMESPACE:.metadata.namespace,NAME:.metadata.name,READY:.status.containerStatuses[*].ready,STATUS:.status.phase,RESTARTS:.status.containerStatuses[*].restartCount,OWNER-KIND:.metadata.ownerReferences[*].kind' | grep -v -E "^(\S+\s+)(\S+\s+)(\S+\s+)(\S+\s+)(\S+\s+)Job" | awk 'NR==1 || /^(\S+\s+)(\S+\s+)\S*?false\S*?\s+/'
}

# kubectl delete pod --all -n ingress-nginx >/dev/null

echo "##########################################################################"

result=$(not_ready_pod_list)
if [ "${result}" != "" ]; then
  echo "Please wait for these pods to be ready..."
  display_not_ready_pod
fi

while [ "$result" != "" ]; do
  sleep 10

  new_result=$(not_ready_pod_list)
  if [ "${new_result}" != "" ]; then
    if [ "${new_result}" != "${result}" ]; then
      echo "##########################################################################"
      echo "Please wait for these pods to be ready..."
      display_not_ready_pod
    fi
  fi
  result=${new_result}
done

helm list -n ${IDO_TEAM} --no-headers=true | awk '{print $1'} > packages.txt
kubectl delete -n ${IDO_TEAM} cm packages-installed --ignore-not-found=true
kubectl create -n ${IDO_TEAM} cm packages-installed --from-file=packages=./packages.txt

echo "Done!"
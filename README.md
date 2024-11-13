### Simple cli tool to help manage kubernetes on google gcloud

Usage:   k8s

		 -cn (change namespace)

		 -cc (change context)

		 -cp (change project)

		 -lc (list contexts)

		 -lp (list google projects)
		 
		 -t (generate token for proxy auth)

Aliases: .bash_profile

		alias A="kubectl get pod -o=custom-columns=NODE:.spec.nodeName,NAME:.metadata.name,NAMESPACE:.metadata.namespace --all-namespaces"

		alias C="kubectl delete pod --field-selector=status.phase==Succeeded --all-namespaces"

		alias J="kubectl get jobs --sort-by=.status.startTime --namespace helix-jobs"

		alias K="kubectl"

		alias L="kubectl logs -f"

		alias N="kubectl get nodes"

		alias S="kubectl get pods --sort-by=.status.startTime"
fooff
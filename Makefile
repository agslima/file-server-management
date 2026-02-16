.PHONY: snyk-report
snyk-report:
	./hack/snyk-report.sh $(target_branch)

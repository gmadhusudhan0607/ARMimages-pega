# workaround for BUG-860056 (.terraform.lock.hcl file is copied to terraform zip)
# created ISSUE-122771 for thinking about better solution
.PHONY: terraform-lock-clean
terraform-lock-clean:
	rm -rf ./build/terraformRun/aws/.terraform*
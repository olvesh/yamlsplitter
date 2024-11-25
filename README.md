# yamlsplitter

This is a little tool created in tight cooperation with claude.ai :-D

It is tuned into splitting the output from the LLM into files, e.g:
```commandline
❯ cat ~/crossplane-test-files.txt | ./yamlsplitter
Created: test/observed-resources.yaml
Created: test/k8s-to-aws-claim.yaml
Created: test/aws-to-k8s-claim.yaml
Created: test/test-render.sh

```

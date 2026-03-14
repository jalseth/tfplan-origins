# tfplan-origins

A utility to map resources in a Terraform plan back to their origin source files.

## Output format

The enriched plan is the standard Terraform plan JSON with a `_loc` field added to each resource change, pointing to the source file and line where the resource is defined:

```json
{
  "resource_changes": [
    {
      "address": "aws_s3_bucket.example",
      "_loc": {
        "file": "infra/main.tf",
        "line": 12
      },
      ...
    }
  ]
}
```

## GitHub Action

The easiest way to use `tfplan-origins` is via the GitHub Action. It builds the tools from source and enriches your Terraform plan JSON with source file locations.

```yaml
- name: 'create plan'
  working-directory: 'infra'
  run: |
    terraform init
    terraform plan -out=plan.bin
    terraform show -json plan.bin > plan.json

- name: 'enrich plan with origins'
  id: 'origins'
  uses: 'jalseth/tfplan-origins@main'
  with:
    terraform-dir: 'infra'
    plan-json: 'infra/plan.json'

- name: 'use enriched plan'
  run: cat '${{ steps.origins.outputs.enriched-plan-file }}'
```

### Inputs

| Input | Required | Default | Description |
|-------|----------|---------|-------------|
| `plan-json` | yes | | Path to the Terraform plan JSON (output of `terraform show -json`) |
| `terraform-dir` | no | `.` | Root Terraform config directory |
| `output-file` | no | `enriched_plan.json` | Path where the enriched plan JSON will be written |
| `go-version` | no | `1.25` | Go version used to build the tools |
| `cache` | no | `true` | Cache the built binaries between runs |

### Outputs

| Output | Description |
|--------|-------------|
| `enriched-plan-file` | Absolute path to the enriched plan JSON |


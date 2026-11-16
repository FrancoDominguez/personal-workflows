# Terraform state: GCS backend

State lives in the `gmail-scanner-512523-tfstate` GCS bucket, not on any
laptop. Every machine that runs terraform reads and writes the same state,
and GCS object locking prevents two concurrent applies from corrupting it.

The bucket itself is **not** managed by terraform — it has to exist before
`terraform init` can run, so it's created once with gcloud (bootstrap
chicken-and-egg). Versioning is enabled so any state mishap can be rolled
back to a previous object version.

## One-time bootstrap

Run from the machine that has the current good `terraform.tfstate`:

```sh
# 1. Create the state bucket (one time ever)
gcloud storage buckets create gs://gmail-scanner-512523-tfstate \
  --project gmail-scanner-512523 \
  --location us-central1 \
  --uniform-bucket-level-access
gcloud storage buckets update gs://gmail-scanner-512523-tfstate --versioning

# 2. From terraform/, with this branch checked out: migrate local state up
terraform init -migrate-state   # answer "yes" to copy local state to GCS

# 3. Sanity check — should show no changes
terraform plan
```

The local `terraform.tfstate` / `terraform.tfstate.backup` files are dead
weight after migration and can be deleted.

## Every other machine

Delete any stale local `terraform.tfstate` first, then from `terraform/`:

```sh
terraform init
```

It connects to the bucket and shares the same state from then on.

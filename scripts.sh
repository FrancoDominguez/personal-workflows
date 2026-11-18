PERSONAL_WORKFLOWS_ROOT=$(cd "$(dirname "${BASH_SOURCE[0]:-$0}")" && pwd)

bndf() {
    local env_file="$PERSONAL_WORKFLOWS_ROOT/.env"
    [[ -f $env_file ]] || { echo "bnd: missing $env_file" >&2; return 1; }

    local repo
    repo=$(grep -E '^GITHUB_REPO=' "$env_file" | cut -d= -f2- | tr -d '"'"'")
    [[ -n $repo ]] || { echo "bnd: GITHUB_REPO not set in $env_file" >&2; return 1; }

    local ref=${1:-$(git -C "$PERSONAL_WORKFLOWS_ROOT" branch --show-current)}
    git -C "$PERSONAL_WORKFLOWS_ROOT" fetch --quiet origin "$ref" || return
    local ahead
    ahead=$(git -C "$PERSONAL_WORKFLOWS_ROOT" rev-list --count "origin/$ref..$ref")
    if (( ahead > 0 )); then
        echo "bnd: $ref is $ahead commit(s) ahead of origin, push first" >&2
        return 1
    fi

    echo "bnd: dispatching Build and Deploy on $repo@$ref"
    gh workflow run publish_functions.yaml --repo "$repo" --ref "$ref" || return
    sleep 5

    local run_id
    run_id=$(gh run list --repo "$repo" --workflow publish_functions.yaml --branch "$ref" \
        --limit 1 --json databaseId --jq '.[0].databaseId')
    [[ -n $run_id ]] || { echo "bnd: could not find the run" >&2; return 1; }
    gh run watch "$run_id" --repo "$repo" --exit-status
}

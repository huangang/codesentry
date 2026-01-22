#!/bin/bash
#
# GitLab Pre-receive Hook for CodeSentry AI Review
#
# Installation (GitLab Self-managed):
#   1. Copy this script to: /opt/gitlab/embedded/service/gitlab-shell/hooks/pre-receive.d/codesentry
#   2. chmod +x /opt/gitlab/embedded/service/gitlab-shell/hooks/pre-receive.d/codesentry
#   3. Configure the variables below
#
# Or for per-project hooks:
#   1. Copy to: <repo>.git/custom_hooks/pre-receive
#   2. chmod +x <repo>.git/custom_hooks/pre-receive

CODESENTRY_URL="${CODESENTRY_URL:-http://localhost:8080}"
CODESENTRY_API_KEY="${CODESENTRY_API_KEY:-}"
TIMEOUT="${CODESENTRY_TIMEOUT:-180}"

while read oldrev newrev refname; do
    if [ "$oldrev" = "0000000000000000000000000000000000000000" ]; then
        range="$newrev"
    else
        range="$oldrev..$newrev"
    fi

    if [ "$newrev" = "0000000000000000000000000000000000000000" ]; then
        continue
    fi

    project_url=$(git config --get remote.origin.url 2>/dev/null || echo "$GL_PROJECT_PATH")
    if [ -z "$project_url" ]; then
        project_url="$GL_REPOSITORY"
    fi

    author=$(git log -1 --format='%an' "$newrev" 2>/dev/null)
    message=$(git log -1 --format='%s' "$newrev" 2>/dev/null)
    diffs=$(git diff "$range" 2>/dev/null)

    if [ -z "$diffs" ]; then
        continue
    fi

    payload=$(jq -n \
        --arg project_url "$project_url" \
        --arg commit_sha "$newrev" \
        --arg ref "$refname" \
        --arg author "$author" \
        --arg message "$message" \
        --arg diffs "$diffs" \
        '{project_url: $project_url, commit_sha: $commit_sha, ref: $ref, author: $author, message: $message, diffs: $diffs}')

    response=$(curl -s -w "\n%{http_code}" \
        --max-time "$TIMEOUT" \
        -X POST \
        -H "Content-Type: application/json" \
        -H "X-API-Key: $CODESENTRY_API_KEY" \
        -d "$payload" \
        "$CODESENTRY_URL/api/review/sync")

    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$d')

    if [ "$http_code" != "200" ]; then
        echo "CodeSentry: Review request failed (HTTP $http_code)"
        echo "$body"
        exit 1
    fi

    passed=$(echo "$body" | jq -r '.passed')
    score=$(echo "$body" | jq -r '.score')
    min_score=$(echo "$body" | jq -r '.min_score')
    msg=$(echo "$body" | jq -r '.message')

    if [ "$passed" != "true" ]; then
        echo ""
        echo "=========================================="
        echo "  CodeSentry AI Review FAILED"
        echo "=========================================="
        echo "  Score: $score / 100"
        echo "  Required: $min_score"
        echo "  $msg"
        echo "=========================================="
        echo ""
        exit 1
    fi

    echo "CodeSentry: Review passed (Score: $score)"
done

exit 0

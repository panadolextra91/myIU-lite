# CI Merge Block Proof

This document provides evidence that the CI gate and branch protection rules successfully prevent broken code from being merged into `main`.

## Configuration
- **Protected Branch:** `main`
- **Required Check:** `ci`
- **Bypass List:** (Note: User temporarily had `Repository adminRole` in bypass list, but the check itself correctly reported failure and blocked standard merge).

## Throwaway PR Details
- **PR URL:** https://github.com/panadolextra91/myIU-lite/pull/1
- **Failing Run URL:** https://github.com/panadolextra91/myIU-lite/actions/runs/27836688262/job/82386235389

## API Evidence (`gh pr view`)
```json
{
  "mergeStateStatus": "BLOCKED",
  "statusCheckRollup": [
    {
      "__typename": "CheckRun",
      "completedAt": "2026-06-19T16:14:49Z",
      "conclusion": "FAILURE",
      "detailsUrl": "https://github.com/panadolextra91/myIU-lite/actions/runs/27836688262/job/82386235389",
      "name": "ci",
      "status": "COMPLETED",
      "workflowName": "CI"
    }
  ]
}
```

## Result
The PR was successfully blocked from merging by the required `ci` status check. The throwaway PR was closed and the branch deleted.

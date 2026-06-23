# Triggering search index update from mal-community / mal-intl

Add this workflow to **both** `chainreactors/mal-community` and `chainreactors/mal-intl`
repos to auto-trigger search index rebuild when plugins are updated:

```yaml
# .github/workflows/notify-malice-network.yaml
name: notify-malice-network

on:
  push:
    branches: [master, main]

jobs:
  trigger:
    runs-on: ubuntu-latest
    steps:
      - name: Trigger search index update
        run: |
          curl -X POST \
            -H "Accept: application/vnd.github+json" \
            -H "Authorization: Bearer ${{ secrets.MALICE_PAT }}" \
            https://api.github.com/repos/chainreactors/malice-network/dispatches \
            -d '{"event_type":"mal-community-updated"}'
```

Set `MALICE_PAT` secret in both repos with a PAT that has `repo` scope
for `chainreactors/malice-network`.

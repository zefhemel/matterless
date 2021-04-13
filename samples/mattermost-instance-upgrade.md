# Detect and notify about changes in Mattermost instance feature flags and versions

# import
* file:lib/mattermost.md

# mattermostInstanceWatcher CommunityDaily
```yaml
url: https://community-daily.mattermost.com
events:
  flag:*:
    - FeatureFlagChange
  upgrade:
    - VersionUpgrade
```

# mattermostInstanceWatcher MasterQA
```yaml
url: https://master.test.mattermost.com
events:
  flag:*:
    - FeatureFlagChange
  upgrade:
    - VersionUpgrade
```

# function FeatureFlagChange
```javascript
function handle(event) {
    console.log("Feature flag change", event);
} 
```

# function VersionUpgrade

```javascript
function handle(event) {
    console.log("Version upgrade", event);
}
```


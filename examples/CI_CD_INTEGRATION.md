# CI/CD Integration Guide

Integrate Hades into your CI/CD pipeline for automated deployments.

## GitHub Actions

### Basic Workflow

```yaml
name: Deploy with Hades

on:
  push:
    tags:
      - 'v*'

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Setup Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'

      - name: Build Application
        run: make build

      - name: Install Hades
        run: |
          git clone https://github.com/SoftKiwiGames/hades /tmp/hades
          cd /tmp/hades
          make build
          sudo mv build/hades /usr/local/bin/

      - name: Setup SSH Key
        run: |
          mkdir -p ~/.ssh
          echo "${{ secrets.DEPLOY_SSH_KEY }}" > ~/.ssh/deploy_key
          chmod 600 ~/.ssh/deploy_key

      - name: Deploy to Staging
        run: |
          hades run staging-deploy \
            -f .hades/hadesfile.yaml \
            -i .hades/inventory.yaml \
            -e VERSION=${{ github.ref_name }} \
            -e COMMIT=${{ github.sha }}

      - name: Deploy to Production
        if: github.ref == 'refs/heads/main'
        run: |
          hades run production-deploy \
            -f .hades/hadesfile.yaml \
            -i .hades/inventory.yaml \
            -e VERSION=${{ github.ref_name }}
```

### With Approval Gate

```yaml
jobs:
  deploy-canary:
    runs-on: ubuntu-latest
    steps:
      - name: Deploy Canary
        run: |
          hades run canary \
            -f hadesfile.yaml \
            -i inventory.yaml \
            -e VERSION=${{ github.ref_name }}

  approve:
    needs: deploy-canary
    runs-on: ubuntu-latest
    environment: production
    steps:
      - name: Manual Approval
        run: echo "Approved"

  deploy-production:
    needs: approve
    runs-on: ubuntu-latest
    steps:
      - name: Deploy Production
        run: |
          hades run production-rollout \
            -f hadesfile.yaml \
            -i inventory.yaml \
            -e VERSION=${{ github.ref_name }}
```

## GitLab CI

```yaml
variables:
  HADES_VERSION: "1.0.0"

stages:
  - build
  - deploy-staging
  - deploy-production

before_script:
  - apt-get update && apt-get install -y openssh-client
  - eval $(ssh-agent -s)
  - echo "$SSH_PRIVATE_KEY" | tr -d '\r' | ssh-add -
  - mkdir -p ~/.ssh && chmod 700 ~/.ssh

install-hades:
  stage: .pre
  script:
    - wget https://github.com/SoftKiwiGames/hades/releases/download/v${HADES_VERSION}/hades
    - chmod +x hades
    - mv hades /usr/local/bin/
  cache:
    paths:
      - /usr/local/bin/hades

deploy-staging:
  stage: deploy-staging
  script:
    - hades run staging-deploy
        -f .hades/hadesfile.yaml
        -i .hades/inventory.yaml
        -e VERSION=$CI_COMMIT_TAG
        -e COMMIT=$CI_COMMIT_SHA
  only:
    - tags

deploy-production:
  stage: deploy-production
  script:
    - hades run production-deploy
        -f .hades/hadesfile.yaml
        -i .hades/inventory.yaml
        -e VERSION=$CI_COMMIT_TAG
  when: manual
  only:
    - tags
```

## Jenkins

```groovy
pipeline {
    agent any

    environment {
        VERSION = "${env.GIT_TAG}"
        COMMIT = "${env.GIT_COMMIT}"
    }

    stages {
        stage('Build') {
            steps {
                sh 'make build'
            }
        }

        stage('Publish Artifact') {
            steps {
                sh '''
                    hades run publish \
                      -f hadesfile.yaml \
                      -i inventory.yaml \
                      -e VERSION=${VERSION}
                '''
            }
        }

        stage('Deploy Canary') {
            steps {
                sh '''
                    hades run canary \
                      -f hadesfile.yaml \
                      -i inventory.yaml \
                      -e VERSION=${VERSION}
                '''
            }
        }

        stage('Approve Production') {
            steps {
                input message: 'Deploy to production?', ok: 'Deploy'
            }
        }

        stage('Deploy Production') {
            steps {
                sh '''
                    hades run production-rollout \
                      -f hadesfile.yaml \
                      -i inventory.yaml \
                      -e VERSION=${VERSION}
                '''
            }
        }
    }

    post {
        failure {
            sh '''
                hades run rollback \
                  -f hadesfile.yaml \
                  -i inventory.yaml \
                  -e VERSION=${PREVIOUS_VERSION}
            '''
        }
    }
}
```

## CircleCI

```yaml
version: 2.1

jobs:
  deploy:
    docker:
      - image: cimg/go:1.21
    steps:
      - checkout
      - run:
          name: Install Hades
          command: |
            git clone https://github.com/SoftKiwiGames/hades
            cd hades && make build
            sudo mv build/hades /usr/local/bin/

      - run:
          name: Setup SSH
          command: |
            mkdir -p ~/.ssh
            echo "$SSH_KEY" | base64 -d > ~/.ssh/deploy_key
            chmod 600 ~/.ssh/deploy_key

      - run:
          name: Deploy
          command: |
            hades run deploy \
              -f .circleci/hadesfile.yaml \
              -i .circleci/inventory.yaml \
              -e VERSION=${CIRCLE_TAG}

workflows:
  deploy:
    jobs:
      - deploy:
          filters:
            tags:
              only: /^v.*/
            branches:
              ignore: /.*/
```

## Best Practices

### 1. Version from Git Tags

```bash
# Extract version from git tag
VERSION=$(git describe --tags --always)

hades run deploy -e VERSION=$VERSION
```

### 2. Secrets Management

**GitHub Actions**:
```yaml
- name: Deploy
  env:
    SSH_KEY: ${{ secrets.DEPLOY_SSH_KEY }}
    DB_PASSWORD: ${{ secrets.DB_PASSWORD }}
  run: |
    hades run deploy -e DB_PASSWORD=$DB_PASSWORD
```

**Store SSH keys as secrets**, not in repository.

### 3. Dry-Run in PR

```yaml
on: pull_request

jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - name: Dry-Run Deploy
        run: |
          hades run deploy \
            -e VERSION=pr-${{ github.event.number }} \
            --dry-run
```

Catches errors before merge.

### 4. Separate Inventories

```
.hades/
  ├── hadesfile.yaml
  ├── inventory-staging.yaml
  ├── inventory-production.yaml
```

```bash
# Staging
hades run deploy -i inventory-staging.yaml

# Production
hades run deploy -i inventory-production.yaml
```

### 5. Build Once, Deploy Many

```yaml
jobs:
  build:
    steps:
      - run: make build
      - run: hades run publish -e VERSION=$TAG

  deploy-staging:
    needs: build
    steps:
      - run: hades run deploy-staging -e VERSION=$TAG

  deploy-production:
    needs: deploy-staging
    steps:
      - run: hades run deploy-production -e VERSION=$TAG
```

Same artifact deployed everywhere.

### 6. Rollback Plan

```yaml
jobs:
  deploy:
    steps:
      - run: hades run deploy -e VERSION=$NEW_VERSION
    on-failure:
      - run: hades run rollback -e VERSION=$PREVIOUS_VERSION
```

Always have rollback ready.

## Dynamic Inventory

### From Cloud Provider

```bash
# AWS
aws ec2 describe-instances \
  --filters "Name=tag:Environment,Values=production" \
  --query 'Reservations[].Instances[].[PrivateIpAddress,Tags[?Key==`Name`].Value]' \
  --output text | \
  awk '{print "  - name: " $2 "\n    addr: " $1 "\n    user: deploy\n    key: ~/.ssh/deploy"}' \
  > inventory.yaml
```

### From Kubernetes

```bash
kubectl get nodes -o json | \
  jq -r '.items[] | "  - name: \(.metadata.name)\n    addr: \(.status.addresses[] | select(.type==\"InternalIP\") | .address)\n    user: root\n    key: ~/.ssh/k8s-key"' \
  > inventory.yaml
```

### From Terraform Output

```hcl
# terraform outputs
output "server_ips" {
  value = aws_instance.app[*].private_ip
}
```

```bash
terraform output -json server_ips | \
  jq -r '.[] | "  - name: app-\(.)\n    addr: \(.)\n    user: deploy\n    key: ~/.ssh/deploy"' \
  > inventory.yaml
```

## Monitoring Integration

### Slack Notifications

```bash
#!/bin/bash
set -e

SLACK_WEBHOOK="https://hooks.slack.com/..."

# Deploy
if hades run deploy -e VERSION=$VERSION; then
  curl -X POST $SLACK_WEBHOOK -d "{\"text\": \"✅ Deploy $VERSION succeeded\"}"
else
  curl -X POST $SLACK_WEBHOOK -d "{\"text\": \"❌ Deploy $VERSION failed\"}"
  exit 1
fi
```

### Datadog Events

```bash
#!/bin/bash

# Before deploy
curl -X POST "https://api.datadoghq.com/api/v1/events" \
  -H "DD-API-KEY: $DD_API_KEY" \
  -d "{
    \"title\": \"Deploy Started\",
    \"text\": \"Version $VERSION deployment started\",
    \"tags\": [\"deployment\", \"hades\"]
  }"

# Deploy
hades run deploy -e VERSION=$VERSION

# After deploy
curl -X POST "https://api.datadoghq.com/api/v1/events" \
  -H "DD-API-KEY: $DD_API_KEY" \
  -d "{
    \"title\": \"Deploy Completed\",
    \"text\": \"Version $VERSION deployed successfully\",
    \"tags\": [\"deployment\", \"hades\"]
  }"
```

## Example Complete Pipeline

```yaml
name: Complete Production Pipeline

on:
  push:
    tags:
      - 'v*'

env:
  VERSION: ${{ github.ref_name }}

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Build
        run: make build
      - name: Upload Artifact
        uses: actions/upload-artifact@v3
        with:
          name: binary
          path: build/app

  publish:
    needs: build
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/download-artifact@v3
      - name: Publish to Registry
        run: |
          hades run publish \
            -f .hades/hadesfile.yaml \
            -i .hades/inventory.yaml \
            -e VERSION=$VERSION

  deploy-canary:
    needs: publish
    runs-on: ubuntu-latest
    steps:
      - name: Deploy Canary
        run: |
          hades run canary \
            -f .hades/hadesfile.yaml \
            -i .hades/inventory.yaml \
            -e VERSION=$VERSION

  verify-canary:
    needs: deploy-canary
    runs-on: ubuntu-latest
    steps:
      - name: Run Tests
        run: ./scripts/verify-canary.sh $VERSION

  approve-production:
    needs: verify-canary
    runs-on: ubuntu-latest
    environment: production
    steps:
      - name: Manual Approval
        run: echo "Approved for production"

  deploy-production:
    needs: approve-production
    runs-on: ubuntu-latest
    steps:
      - name: Deploy Production
        run: |
          hades run production-rollout \
            -f .hades/hadesfile.yaml \
            -i .hades/inventory.yaml \
            -e VERSION=$VERSION

      - name: Notify Success
        if: success()
        run: |
          curl -X POST ${{ secrets.SLACK_WEBHOOK }} \
            -d '{"text": "✅ Production deploy ${{ env.VERSION }} succeeded"}'

      - name: Notify Failure
        if: failure()
        run: |
          curl -X POST ${{ secrets.SLACK_WEBHOOK }} \
            -d '{"text": "❌ Production deploy ${{ env.VERSION }} failed"}'
```

## Security Considerations

1. **SSH Keys**: Store as encrypted secrets, never in code
2. **Inventory**: Don't commit sensitive data, generate dynamically
3. **Secrets**: Use CI/CD secret management, not env files
4. **Audit Logs**: Capture Hades output for compliance
5. **Least Privilege**: SSH user should have minimal permissions

## Troubleshooting CI/CD

**SSH Connection Issues**:
```bash
# Debug SSH in CI
ssh -vvv -i $SSH_KEY user@host
```

**Environment Variables Not Set**:
```bash
# Print all env vars in CI
env | sort
```

**Hades Not Found**:
```bash
# Verify installation
which hades
hades --version
```

**Timeout Issues**:
```bash
# Increase SSH timeout
export SSH_TIMEOUT=300
```

## Next Steps

- Implement blue-green deployments
- Add automated rollback on failure
- Integrate health checks between stages
- Set up deployment dashboards
- Add deployment metrics collection

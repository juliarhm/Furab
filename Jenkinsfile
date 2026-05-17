pipeline {
    agent any

    environment {
        DOCKER_REGISTRY = 'registry.furab.io'
        KUBE_NAMESPACE  = 'furab'
        GO_VERSION      = '1.22'
    }

    stages {
        // Stage 1: Checkout repository
        stage('Checkout') {
            steps {
                checkout scm
                powershell 'Write-Host "Checked out branch: $env:GIT_BRANCH"'
            }
        }

        // Stage 2: Unit Tests (no DB, no external services)
        stage('Unit Tests') {
            steps {
                powershell '''
                    Set-Location furab-backend
                    Write-Host "Running unit tests..."
                    $services = Get-ChildItem -Path services -Directory
                    foreach ($s in $services) {
                        Write-Host "=== Unit Testing: $($s.Name) ==="
                        Set-Location $s.FullName
                        go test ./test/unit/... -v -cover -coverprofile=coverage.out
                        if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
                        Set-Location ../..
                    }
                '''
            }
            post {
                always {
                    powershell 'Write-Host "Unit test results published"'
                }
            }
        }

        // Stage 3: Lint & Vet (static analysis)
        stage('Lint/Vet') {
            steps {
                powershell '''
                    Set-Location furab-backend
                    Write-Host "Running go vet..."
                    $services = Get-ChildItem -Path services -Directory
                    foreach ($s in $services) {
                        Write-Host "=== Vetting: $($s.Name) ==="
                        Set-Location $s.FullName
                        go vet ./...
                        Set-Location ../..
                    }
                '''
            }
        }

        // Stage 4: Build Docker Images (local)
        stage('Build Image') {
            steps {
                powershell '''
                    Set-Location furab-backend
                    Write-Host "Building Docker images..."
                    $services = Get-ChildItem -Path services -Directory
                    foreach ($s in $services) {
                        Write-Host "=== Building: $($s.Name) ==="
                        docker build -t "$env:DOCKER_REGISTRY/$($s.Name):$env:BUILD_NUMBER" -t "$env:DOCKER_REGISTRY/$($s.Name):latest" -f "$($s.FullName)/Dockerfile" .
                        if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
                    }
                '''
            }
        }

        // Stage 5: Functional Tests (requires DB and infrastructure)
        stage('Functional Tests') {
            steps {
                powershell '''
                    Set-Location furab-backend
                    Write-Host "Starting test infrastructure sequentially to prevent Docker crash..."
                    Write-Host "Starting Postgres..."
                    docker compose -f deploy/docker/docker-compose.yml up -d postgres
                    Start-Sleep -Seconds 5
                    
                    Write-Host "Starting Redis..."
                    docker compose -f deploy/docker/docker-compose.yml up -d redis
                    Start-Sleep -Seconds 5
                    
                    Write-Host "Starting RabbitMQ..."
                    docker compose -f deploy/docker/docker-compose.yml up -d rabbitmq
                    Start-Sleep -Seconds 5
                    
                    Write-Host "Starting Kafka (and Zookeeper)..."
                    docker compose -f deploy/docker/docker-compose.yml up -d kafka
                    
                    Write-Host "Waiting for all services to be fully ready..."
                    Start-Sleep -Seconds 15

                    Write-Host "Running functional tests..."
                    $services = Get-ChildItem -Path services -Directory
                    foreach ($s in $services) {
                        Write-Host "=== Functional Testing: $($s.Name) ==="
                        Set-Location $s.FullName
                        go test ./test/functional/... -v -tags=functional
                        if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
                        Set-Location ../..
                    }
                '''
            }
            post {
                always {
                    powershell 'Set-Location furab-backend; docker compose -f deploy/docker/docker-compose.yml down'
                }
            }
        }

        // Stage 6: Push Docker Images to Registry
        stage('Push Image') {
            when {
                branch 'main'
            }
            steps {
                powershell '''
                    Set-Location furab-backend
                    Write-Host "Pushing Docker images..."
                    $services = Get-ChildItem -Path services -Directory
                    foreach ($s in $services) {
                        Write-Host "=== Pushing: $($s.Name) ==="
                        docker push "$env:DOCKER_REGISTRY/$($s.Name):$env:BUILD_NUMBER"
                        docker push "$env:DOCKER_REGISTRY/$($s.Name):latest"
                    }
                '''
            }
        }

        // Stage 7: Deploy to Kubernetes
        stage('Deploy to Kubernetes') {
            when {
                branch 'main'
            }
            steps {
                powershell '''
                    Set-Location furab-backend
                    Write-Host "Deploying to Kubernetes..."
                    kubectl apply -f deploy/kubernetes/namespace.yaml

                    helm upgrade --install furab deploy/helm/furab-chart/ --namespace $env:KUBE_NAMESPACE --set image.tag=$env:BUILD_NUMBER --wait --timeout 300s
                '''
            }
        }

        // Stage 8: Verify Deployment
        stage('Verify') {
            when {
                branch 'main'
            }
            steps {
                powershell '''
                    Write-Host "Verifying deployment..."
                    Start-Sleep -Seconds 10

                    $services = @("auth-service", "otp-service", "user-service", "driver-service", "ride-order-service", "food-order-service", "cart-service", "matching-service", "payment-service", "wallet-service", "settlement-service", "pricing-service", "promo-service", "location-service", "chat-service", "notification-service", "email-service", "emergency-service", "merchant-service", "menu-service", "rating-service", "review-service", "audit-log-service")

                    foreach ($s in $services) {
                        Write-Host "=== Checking: $s ==="
                        kubectl rollout status deployment/$s -n $env:KUBE_NAMESPACE --timeout=120s
                    }
                '''
            }
        }
    }

    post {
        success {
            echo 'Pipeline completed successfully!'
        }
        failure {
            echo 'Pipeline failed!'
        }
        always {
            cleanWs()
        }
    }
}

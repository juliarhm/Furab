$dirs = Get-ChildItem -Path d:\Pekerjaan\furabapps\furab-backend\services -Directory
$failed = @()

foreach ($dir in $dirs) {
    Write-Host "Testing $($dir.Name)..."
    
    # Run tests directly
    $output = rtk go test -tags=functional "./services/$($dir.Name)/test/functional/..." 2>&1
    
    if ($LASTEXITCODE -ne 0) {
        Write-Host "FAILED: $($dir.Name)"
        $failed += $dir.Name
    } else {
        Write-Host "PASSED: $($dir.Name)"
    }
}

if ($failed.Count -eq 0) {
    Write-Host "=============================="
    Write-Host "ALL TESTS PASSED SUCCESSFULLY!"
    Write-Host "=============================="
} else {
    Write-Host "=============================="
    Write-Host "FAILED SERVICES: $($failed -join ', ')"
    Write-Host "=============================="
}

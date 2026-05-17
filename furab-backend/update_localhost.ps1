$files = Get-ChildItem -Path d:\Pekerjaan\furabapps\furab-backend\services -Filter *functional_test.go -Recurse
foreach ($f in $files) {
    $content = Get-Content $f.FullName
    if ($content -match 'localhost') {
        $content = $content -replace 'localhost', '127.0.0.1'
        Set-Content -Path $f.FullName -Value $content
        Write-Host "Updated $($f.FullName)"
    }
}
Write-Host "Done replacing localhost."

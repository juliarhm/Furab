$files = Get-ChildItem -Path d:\Pekerjaan\furabapps\furab-backend\services -Filter Dockerfile -Recurse
foreach ($file in $files) {
    $content = Get-Content $file.FullName
    $content = $content -replace 'golang:1.22-alpine', 'golang:alpine'
    Set-Content -Path $file.FullName -Value $content
}
Write-Host "Done replacing."

$services = Get-ChildItem -Path d:\Pekerjaan\furabapps\furab-backend\services -Directory
foreach ($s in $services) {
    Write-Host "Tidying module in $($s.Name)..."
    Set-Location $s.FullName
    Remove-Item -ErrorAction SilentlyContinue go.sum
    go mod tidy
}
Set-Location d:\Pekerjaan\furabapps\furab-backend
Write-Host "Syncing workspace..."
go work sync
Write-Host "Workspace synchronized!"

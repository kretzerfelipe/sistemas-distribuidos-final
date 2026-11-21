$ErrorActionPreference = "Stop"
$h = "http://localhost:8080"
$j = "application/json"

$adminTok = (Invoke-RestMethod -Method Post "$h/auth/login" -ContentType $j -Body '{"email":"admin@futhub.local","password":"admin12345"}').accessToken
$adm = @{Authorization = "Bearer $adminTok"}

try { Invoke-RestMethod -Method Post "$h/auth/register" -ContentType $j -Body '{"name":"Dono","email":"team@example.com","password":"s3cret-pass","accountType":"TEAM"}' | Out-Null } catch {}
$teamTok = (Invoke-RestMethod -Method Post "$h/auth/login" -ContentType $j -Body '{"email":"team@example.com","password":"s3cret-pass"}').accessToken
$tm = @{Authorization = "Bearer $teamTok"}

$s = Get-Random -Maximum 99999
$t1 = Invoke-RestMethod -Method Post "$h/teams" -Headers $tm -ContentType $j -Body (@{name = "Time A $s"; city = "Blumenau"} | ConvertTo-Json)
$t2 = Invoke-RestMethod -Method Post "$h/teams" -Headers $tm -ContentType $j -Body (@{name = "Time B $s"; city = "Itajai"} | ConvertTo-Json)
$ch = Invoke-RestMethod -Method Post "$h/championships" -Headers $adm -ContentType $j -Body (@{name = "Copa Teste $s"; slotsTotal = 10} | ConvertTo-Json)

foreach ($t in @($t1, $t2)) {
    $body = @{teamId = $t.id} | ConvertTo-Json
    Invoke-RestMethod -Method Post "$h/championships/$($ch.id)/teams" -Headers $adm -ContentType $j -Body $body | Out-Null
}
Invoke-RestMethod -Method Post "$h/championships/$($ch.id)/start" -Headers $adm | Out-Null

$body = @{championshipId = $ch.id; homeTeamId = $t1.id; awayTeamId = $t2.id; scheduledAt = "2027-03-01T15:00:00Z"} | ConvertTo-Json
$m = Invoke-RestMethod -Method Post "$h/matches" -Headers $adm -ContentType $j -Body $body

"TOKEN=$adminTok"
"MATCH=$($m.id)"
"HOME=$($t1.id)"

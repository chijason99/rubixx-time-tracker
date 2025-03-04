function Get-AuthInfo{
    param (
        [Parameter(Mandatory=$true)]
        [string]$username,
        [Parameter(Mandatory=$true)]
        [securestring]$password
    )
    $password_plain = [Runtime.InteropServices.Marshal]::PtrToStringAuto([Runtime.InteropServices.Marshal]::SecureStringToBSTR($password))
    $username_encoded = [System.Net.WebUtility]::UrlEncode($username)
    $password_encoded = [System.Net.WebUtility]::UrlEncode($password_plain)

    try{
        $time_trak_go_auth_url = "https://rubixx.timetrakgo.com/api/auth/authenticate?username=$username_encoded&password=$password_encoded"

        $auth_response = Invoke-WebRequest -Uri $time_trak_go_auth_url -Method Get
    
        $auth_response_json = $auth_response.Content | ConvertFrom-Json
    
        return [PSCustomObject]@{
            Token = $auth_response_json.user.token
            UserId = $auth_response_json.user.userId
        }
    }
    catch {
        Write-Error "Failed to authenticate"
        throw
    }
}

function Get-StartAndEndOfMonth{
    $start_of_month = Get-Date -Day 1 -Hour 0 -Minute 0 -Second 0
    $end_of_month = $start_of_month.AddMonths(1).AddSeconds(-1)

    $start_of_month_formatted = $start_of_month.ToString("yyyy-MM-dd")
    $end_of_month_formatted = $end_of_month.ToString("yyyy-MM-dd")

    return [PSCustomObject]@{
        StartOfMonth = $start_of_month_formatted
        EndOfMonth = $end_of_month_formatted
    }
}

function Get-CalculatedHours {
    param (
        [string]$user_id,
        [string]$auth_token,
        [string]$start_date,
        [string]$end_date
    )

    try{
        $time_trak_go_time_url = "https://rubixx.timetrakgo.com/api/punch/GetCalculatedHours?userId=$user_id&groupId=&StartDateTime=$start_date&EndDateTime=$end_date&ReturnWorkWeek=false"
        $time_record_response = Invoke-WebRequest -Uri $time_trak_go_time_url -Method Get -Headers @{Authorization = "Bearer $auth_token"}
        $time_record_response_json = $time_record_response.Content | ConvertFrom-Json
    
        return [PSCustomObject]@{
            Hours = $time_record_response_json.userCalculatedData.hours
            TotalHours = $time_record_response_json.userCalculatedData.totalHours.amount
        }
    }
    catch{
        Write-Error "Failed to get calculated hours"
        throw
    }
}

function Calculate-Hours {
    param (
        [array]$hours,
        [float]$total_hours
    )
    $expected_hours_per_day = 7.4
    $expected_hours = $hours.Length * $expected_hours_per_day
    $delta = [Math]::Round($total_hours - $expected_hours, 2)
    $absolute_delta = [Math]::Abs($delta)

    Write-Host "Expected hours: $expected_hours"
    Write-Host "Total hours: $total_hours"

    if ($delta -gt 0) {
        Write-Host "Over by: $absolute_delta hours" -BackgroundColor Green
    } elseif ($delta -lt 0) {
        Write-Host "Under by: $absolute_delta hours" -BackgroundColor Red
    } else {
        Write-Host "Exact hours worked"
    }
}

try{
    $username = Read-Host "Enter your username"
    $password = Read-Host "Enter your password" -AsSecureString

    $user = Get-AuthInfo $username $password
    $dates = Get-StartAndEndOfMonth
    $hours = Get-CalculatedHours -user_id $user.UserId -auth_token $user.Token -start_date $dates.StartOfMonth -end_date $dates.EndOfMonth
    
    Calculate-Hours -hours $hours.Hours -total_hours $hours.TotalHours
}
catch {
    Write-Error $_.Exception.Message
    throw
}
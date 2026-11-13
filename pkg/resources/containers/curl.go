package containers

const curlCommand = "curl " +
	// Retry several times in case the runner pods are not ready yet, or the service is not reachable.
	// Limit the delay between retries to speed recovery: the default retry
	// delay has an exponential backoff, which can take a very long time with
	// many retries.
	// Also limit the total time of a request to reject connection timeouts earlier.
	"--retry 30 --retry-delay 2 --max-time 10 --retry-connrefused " +
	"-X PATCH -H 'Content-Type: application/json' " +
	"http://%s/v1/status " +
	"-d '%s' -s " +
	"-w '\n{\"http_code\":%%{http_code},\"time_total\":%%{time_total},\"time_starttransfer\":%%{time_starttransfer},\"url\":\"%%{url_effective}\",\"remote_ip\":\"%%{remote_ip}\",\"errormsg\":\"%%{errormsg}\", \"retries\":%%{num_retries}}'"

#curl localhost:8080/api/v1/apilist/getApiDetailByApiId/1
#curl localhost:8080/config
curl localhost:8080/config  -X POST -d @data.json --header "Content-Type: application/json"

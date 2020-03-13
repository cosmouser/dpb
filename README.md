# dpb
HTTP proxy for doing LDAP search requests

```
Usage of ./dpb:
  -apiKeyFile string
    	Path to api key file. (default "./apikeys.txt")
  -baseDN string
    	Base DN for search requests. (default "ou=People,dc=example,dc=edu")
  -ldapURI string
    	Full uri path to ldap server for lookups. (default "ldaps://ldap-server.example.edu")
  -port string
    	Port number to listen on (default "9090")
```

## API Key file format example

```
# Comment: Cosmo's key on the line below...
li2u3fn394nrwviabgnap9b789hp39aynpr9
```

## Sending requests

Requests must be sent with a valid x-api-key header. The server accepts POST requests with json in the shape below:

```
{
    "filter": "(|(uid=cosmo)(uid=sammy))",
    "attributeNames": [
        "cn",
        "title",
        "mail",
        "telephoneNumber"
    ]
}
```

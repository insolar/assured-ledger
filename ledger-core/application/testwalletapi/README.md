#Description of TestWalletAPI

It's http API. It accepts POST requests. Parameters are expected from POST body in JSON format.  
Different method are divided by different locations.

#Errors
If invalid JSON is received or not all parameters are given, API returns "Bad request"( http code 400 ).  
If any internal error happens it will be returned in field "error". The structure of the value will be declared later.   

#Minimum output
```
{
    "error": null,
    "traceID": "144b8072ca1d141a3e8d1b81ac8e5888"
}
```

#Examples
* Create wallet  
`curl "127.0.0.1:32301/wallet/create"`
* Transfer amount from one wallet to another  
`curl  -X POST -d '{"from":"testFROM", "to": "testTO", "amount": 100}'  "127.0.0.1:32301/wallet/transfer"`
* Get balance of given wallet
`curl -X POST -d '{ "walletRef": "testRef" }'  "127.0.0.1:32304/wallet/get_balance"`
* Add amount to given wallet
`curl -X POST -d '{ "to": "testRef", "amount": 10 }'  "127.0.0.1:32304/wallet/add_amount"`

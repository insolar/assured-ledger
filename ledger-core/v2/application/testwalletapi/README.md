#Description of TestWalletAPI

It's http API. It accepts POST requests.  
Different method are divided by different locations. Parameters are expected from POST body.  

#Examples
* Create wallet  
`curl "127.0.0.1:32301/wallet/create"`
* Transfer amount from one wallet to another  
`curl  -X POST -d '{"from":"testFROM", "to": "testTO", "amount": 100}'  "127.0.0.1:32301/wallet/transfer"`
* Get balance of given wallet
`curl -X POST -d '{ "walletRef": "testRef" }'  "127.0.0.1:32304/wallet/get_balance"`
* Add amount to given wallet
`curl -X POST -d '{ "to": "testRef", "amount": 10 }'  "127.0.0.1:32304/wallet/add_amount"`

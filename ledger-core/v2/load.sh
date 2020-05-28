
numWallets=50
numTransfers=3
amount=10000
URL=127.0.0.1:32304/wallet

echo "numWallets: $numWallets"
echo "numTransfers: $numTransfers"
echo "amount: $amount"
echo "URL: $URL"

refs=

createWallets()
{
  echo
  echo "Wallets creating starts ..."
  start=`date +%s`
  refs=$(
  for i in `seq 1 $numWallets`
  do
    curl "$URL/create"  2>/dev/null| python -m json.tool
    if [ "$?" != "0" ]
    then
      echo "can't create"
      exit 1
    fi
  done | jq -r .reference
  )
  end=`date +%s`
  echo "Wallets have been created: $((end-start)) seconds"
}

addMoney()
{
  echo
  echo "Money adding starts ..."
  start=`date +%s`
  for ref in $refs
  do
    curl -X POST -d '{ "to": "'$ref'", "amount": '$amount' }'  "$URL/add_amount"
    if [ "$?" != "0" ]
    then
      echo "can't add_amount"
      exit 1
    fi
  done &> /dev/null
  end=`date +%s`
  echo "Money has been added: $((end-start)) seconds"
}

transferMoney()
{
  echo
  echo "Money transferring starts ..."
  start=`date +%s`
  for i in `seq 1 $numTransfers`
  do
    refs=$( echo "$refs" | sort -R )
    for wn in `seq 2 2 $numWallets`
    do
      from=$( echo -e "$refs" | head -n $wn | tail -n 1 )
      toNum=$(( wn - 1 ))
      to=$( echo -e "$refs" | head -n $toNum | tail -n 1 )
      curl  -X POST -d '{"from":"'$from'", "to": "'$to'", "amount": 100}'  "$URL/transfer" &> /dev/null
      if [ "$?" != "0" ]
      then
        echo "can't transfer"
        exit 1
      fi
    done
    echo "... round $i/$numTransfers"
  done
  end=`date +%s`
  echo "Money transferring completed: $((end-start)) seconds"
}

createWallets
addMoney
transferMoney

import "@stdlib/deploy";
import "./bet.tact";
message(0x297012ec) CreateBet {
	id: Int as int128;
}
contract BetMaster {
  const MinTonForStorage: Int = ton("0.01");
  id: Int as int32 = 0;

  init(id: Int as int32) {
    self.id = id;
  }

  receive() {
    cashback(sender());
  }

  receive(msg: CreateBet) {
	let init = initOf Bet(msg.id);
	let betAddress = contractAddress(init);
	let deployValue = context().value - self.MinTonForStorage;
	require(deployValue > 0, "Not enough value to deploy");

	send(SendParameters{
	  to: betAddress,
	  value: deployValue,
	  bounce: true,
	  mode: SendIgnoreErrors,
	  code: init.code,
	  data: init.data,
	  body: Deposit{player: sender()}.toCell(),
	});
  }
} 

import "@stdlib/deploy";

message Winner {
    address: Address;
}

contract Bet with Deployable {
  const MinTonForStorage: Int = ton("0.01"); 
  firstPlayer: Address;
  secondPlayer: Address?;
  amount: Int;
  status: Int;
  id: Int as int32 = 0;

  init(id: Int) {
    self.firstPlayer  = sender();
    self.secondPlayer = null;
    self.amount       = 0;
    self.status       = 0;
    self.id           = id;
  }

  /// Первый игрок депонирует ставку
  receive("deposit") {
    require(self.status == 0, "Already deposited or finished");
    require(context().value > 0, "Amount must be > 0");

    self.amount = context().value;
    self.status = 1;
  }

  /// Второй игрок принимает пари
  receive("accept") {
    require(self.status == 1, "Bet not in NEW status");
    require(context().value == self.amount, "Wrong amount");

    self.secondPlayer = sender();
    self.status       = 2;
  }

  /// Первый игрок отменяет пари до того, как оно принято
  receive("refund") {
    require(self.status == 1, "Cannot refund now");
    // возвращаем всю сумму первому игроку
    message(MessageParameters{
            to: self.firstPlayer,
            mode: SendRemainingValue | SendIgnoreErrors,
            bounce: true,
            body: "Refund".asComment(),
            value: myBalance() - context().value - self.MinTonForStorage,
        });
    self.status = 3;
  }

  /// Обычный метод (не receive) для завершения пари и выплаты выигрыша
  receive(msg: Winner) {
    require(self.status == 2, "Bet not accepted");
    require(msg.address == self.firstPlayer || msg.address == self.secondPlayer, "Invalid winner");

    send(SendParameters{
            to: msg.address,
            bounce: true,
            value: myBalance() - context().value - self.MinTonForStorage,
            mode: SendRemainingValue + SendIgnoreErrors
        });
    self.status = 3;
  }

  /// Служебные геттеры
  get fun status(): Int {
    return self.status;
  }

  get fun amount(): Int {
    return self.amount;
  }

  get fun id(): Int {
    return self.id;
  }
}

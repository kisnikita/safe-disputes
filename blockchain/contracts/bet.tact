import "@stdlib/deploy";

message Winner {
    address: Address;
}

contract Bet with Deployable {
  const MinTonForStorage: Int = ton("0.01"); 
  firstPlayer: Address?;
  secondPlayer: Address?;
  amount: Int;
  status: Int;
  id: Int as int32 = 0;
  firstClaimed: Bool;
  secondClaimed: Bool;

  init(id: Int) {
    self.firstPlayer  = null;
    self.secondPlayer = null;
    self.amount       = 0;
    self.status       = 0;
    self.id           = id;
    self.firstClaimed = false;
    self.secondClaimed = false;
  }

  /// Первый игрок депонирует ставку
  receive("deposit") {
    require(self.status == 0, "Already deposited or finished");
    require(context().value > 0, "Amount must be > 0");

    self.firstPlayer = sender();
    self.amount = context().value;
    self.status = 1;
  }

  /// Второй игрок принимает пари
  receive("accept") {
    require(self.status == 1, "Bet not in NEW status");
    require(context().value == self.amount, "Wrong amount");

    self.secondPlayer = sender();
    self.amount       += context().value;
    self.status       = 2;
  }

  /// Первый игрок отменяет пари до того, как оно принято
  receive("refund") {
    require(self.status == 1, "Cannot refund now");
    // возвращаем всю сумму первому игроку
    message(MessageParameters{
            to: self.firstPlayer!!,
            mode: SendRemainingBalance | SendDestroyIfZero,
            bounce: true,
            value: 0, // because of mode
        });
    self.status = 3;
  }

  /// Ничья: каждый игрок клеймит свою часть отдельно
  receive("draw") {
    require(self.status == 2, "Bet not accepted");
    require(sender() == self.firstPlayer || sender() == self.secondPlayer, "Not a participant");

    let isFirst = sender() == self.firstPlayer;

    if (isFirst) {
      require(!self.firstClaimed, "Already claimed");
    } else {
      require(!self.secondClaimed, "Already claimed");
    }

    let val = (myBalance() - context().value - self.MinTonForStorage) / 2;
    let mode = SendRemainingValue;
    if (self.firstClaimed || self.secondClaimed) {
      val = 0; // because of mode
      mode = SendRemainingBalance | SendDestroyIfZero;
    }
    
    send(SendParameters{
            to: sender(),
            bounce: true,
            value: val,
            mode: mode
        });

    if (isFirst) {
      self.firstClaimed = true;
    } else {
      self.secondClaimed = true;
    }

    if (self.firstClaimed && self.secondClaimed) {
      self.status = 3;
    }
  }

  /// Обычный метод (не receive) для завершения пари и выплаты выигрыша
  receive(msg: Winner) {
    require(self.status == 2, "Bet not accepted");
    require(msg.address == self.firstPlayer || msg.address == self.secondPlayer, "Invalid winner");

    send(SendParameters{
            to: msg.address,
            bounce: true,
            value: 0, // because of mode
            mode: SendRemainingBalance | SendDestroyIfZero
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

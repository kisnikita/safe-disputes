import { BetMaster, CreateBet } from "../../../../blockchain/wrappers/BetMaster";
import { Address, OpenedContract, Sender, toNano } from "@ton/core";
import { useAsyncInitialize } from "./useAsyncInitialize";
import { useTonClient } from "./useTonClient";
import { useTonConnect } from "./useTonConnect";

export function useBetMasterContract() {
    const { client } = useTonClient();
    const { sender, sendSignedTransaction } = useTonConnect();

    const masterContract = useAsyncInitialize(async () => {
        if (!client) return;
        const masterAddress = "EQAYCTpvBNLWfQcdOxRkPgPXpie8ovKofoaG3yIWkFeXHPMC";
        if (!masterAddress) return;

        const contract = BetMaster.fromAddress(Address.parse(masterAddress));
        return client.open(contract) as OpenedContract<BetMaster>;
    }, [client]);

    return {
        createBetWithDeposit: async (betID: bigint, depositValue: string) => {
            if (!masterContract) throw new Error("master contract not ready");
            const msg: CreateBet = {
                $$type: "CreateBet",
                id: betID,
            };
            const totalValue = toNano(depositValue);

            let boc = "";
            const senderWithBocCapture: Sender = {
                address: sender.address,
                send: async (args) => {
                    boc = await sendSignedTransaction(args);
                },
            };

            await masterContract.send(senderWithBocCapture, { value: totalValue }, msg);
            if (!boc) {
                throw new Error("wallet did not return signed boc");
            }

            return boc;
        },
    };
}

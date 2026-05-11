import { BetMaster, CreateBet } from "../../../../blockchain/wrappers/BetMaster";
import { Address, OpenedContract } from "@ton/core";
import { useAsyncInitialize } from "./useAsyncInitialize";
import { useTonClient } from "./useTonClient";
import { useTonConnect } from "./useTonConnect";

export function useBetMasterContract() {
    const { client } = useTonClient();
    const { sendWithBoc } = useTonConnect();

    const masterContract = useAsyncInitialize(async () => {
        if (!client) return;
        const masterAddress = "EQC9siZ7Ss9MZVqU1ywih597rSaekr7gKvWxYmuJNkj2q7Il";
        if (!masterAddress) return;

        const contract = BetMaster.fromAddress(Address.parse(masterAddress));
        return client.open(contract) as OpenedContract<BetMaster>;
    }, [client]);

    return {
        createBetWithDeposit: async (betID: bigint, totalValueNano: bigint, resultDeadlineUnix: number) => {
            if (!masterContract) throw new Error("master contract not ready");
            const msg: CreateBet = {
                $$type: "CreateBet",
                id: betID,
                resultDeadline: BigInt(resultDeadlineUnix),
            };

            return sendWithBoc(async (senderWithBoc) => {
                await masterContract.send(senderWithBoc, { value: totalValueNano }, msg);
            });
        },
        getMinDeposit: async (): Promise<bigint | undefined> => {
            return await masterContract?.getMinDeposit();
        },
        getMinStake: async (): Promise<bigint | undefined> => {
            return await masterContract?.getMinStake();
        },
    };
}

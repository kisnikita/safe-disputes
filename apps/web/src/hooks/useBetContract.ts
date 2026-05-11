import { Accept, Bet, Claim, Cancel, VoteResult } from "../../../../blockchain/wrappers/Bet";
import { Address, OpenedContract, Sender, toNano } from "@ton/core";
import { useTonClient } from "./useTonClient";
import { useTonConnect } from "./useTonConnect";

export function useBetContract() {
    const { client } = useTonClient();
    const { address, sendWithBoc } = useTonConnect();

    const openBet = async (contractAddress: string): Promise<OpenedContract<Bet>> => {
        if (!client) throw new Error("ton client not ready");
        const contract = Bet.fromAddress(Address.parse(contractAddress));
        return client.open(contract) as OpenedContract<Bet>;
    };

    return {
        getAddress: async (
            betID: bigint,
            resultDeadlineUnix: number,
        ) => {
            if (!address) throw new Error("wallet address not found");
            return (await Bet.fromInit(
                betID,
                Address.parse(address),
                BigInt(resultDeadlineUnix),
            )).address;
        },
        accept: async (contractAddress: string): Promise<string> => {
            const betContract = await openBet(contractAddress);
            const stakeWithDeposit = await betContract.getStakeWithDeposit();
            const msg: Accept = { $$type: "Accept" };
            return sendWithBoc(async (senderWithBoc: Sender) => {
                await betContract.send(senderWithBoc, { value: stakeWithDeposit }, msg);
            });
        },
        claim: async (contractAddress: string): Promise<string> => {
            const betContract = await openBet(contractAddress);
            const msg: Claim = { $$type: "Claim" };
            return sendWithBoc(async (senderWithBoc: Sender) => {
                await betContract.send(senderWithBoc, { value: toNano('0.02') }, msg);
            });
        },
        cancel: async (contractAddress: string): Promise<string> => {
            const betContract = await openBet(contractAddress);
            const msg: Cancel = { $$type: "Cancel" };
            return sendWithBoc(async (senderWithBoc: Sender) => {
                await betContract.send(senderWithBoc, { value: toNano('0.02') }, msg);
            });
        },
        vote: async (contractAddress: string, isWin: boolean): Promise<string> => {
            const betContract = await openBet(contractAddress);
            const msg: VoteResult = { $$type: "VoteResult", result: isWin ? 1n : 0n };
            return sendWithBoc(async (senderWithBoc: Sender) => {
                await betContract.send(senderWithBoc, { value: toNano("0.02") }, msg);
            });
        },
        getInvestigationAddress: async (contractAddress: string) => {
            const betContract = await openBet(contractAddress);
            return await betContract.getInvestigationAddress();
        },
    };
}

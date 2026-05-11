import { Investigation, JurorVote, ProvideEvidence } from "../../../../blockchain/wrappers/Investigation";
import { Address, OpenedContract, Sender, toNano } from "@ton/core";
import { useTonClient } from "./useTonClient";
import { useTonConnect } from "./useTonConnect";

export function useInvestigationContract() {
    const { client } = useTonClient();
    const { sendWithBoc } = useTonConnect();

    const openInvestigation = async (contractAddress: string): Promise<OpenedContract<Investigation>> => {
        if (!client) throw new Error("ton client not ready");
        const contract = Investigation.fromAddress(Address.parse(contractAddress));
        return client.open(contract) as OpenedContract<Investigation>;
    };

    return {
        provideEvidence: async (contractAddress: string, hash: bigint): Promise<string> => {
            const inv = await openInvestigation(contractAddress);
            const msg: ProvideEvidence = { $$type: "ProvideEvidence", hash };
            return sendWithBoc(async (senderWithBoc: Sender) => {
                await inv.send(senderWithBoc, { value: toNano("0.02") }, msg);
            });
        },
        vote: async (contractAddress: string, option: 1 | 2 | 3): Promise<string> => {
            const inv = await openInvestigation(contractAddress);
            const msg: JurorVote = { $$type: "JurorVote", option: BigInt(option) };
            return sendWithBoc(async (senderWithBoc: Sender) => {
                await inv.send(senderWithBoc, { value: toNano("0.02") }, msg);
            });
        },
    };
}

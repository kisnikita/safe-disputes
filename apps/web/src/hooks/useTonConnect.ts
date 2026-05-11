import { useTonConnectUI, useTonWallet } from "@tonconnect/ui-react";
import { SenderArguments, Sender, Address } from "@ton/core";

export function useTonConnect(): {
    address: string | null;
    connected: boolean;
    sendWithBoc: (fn: (senderWithBoc: Sender) => Promise<void>) => Promise<string>;
} {
    const [tonConnectUI] = useTonConnectUI()
    const wallet = useTonWallet()
    const senderAddress = wallet?.account?.address ? Address.parse(wallet.account.address) : undefined

    const sendSignedTransaction = async (args: SenderArguments): Promise<string> => {
        if (!wallet?.account?.address) {
            throw new Error("wallet address not found");
        }
        const result = await tonConnectUI.sendTransaction(
            {
                from: wallet.account.address,
                network: wallet.account.chain,
                messages: [
                    {
                        address: args.to.toString(),
                        amount: args.value.toString(),
                        payload: args.body?.toBoc().toString("base64"),
                    },
                ],
                validUntil: Date.now() + 5 * 60 * 1000, // 5 minutes for user to approve
            },
            {
                notifications: []
            }
        );

        if (!result?.boc) {
            throw new Error("wallet did not return signed boc");
        }

        return result.boc;
    };
    const sendWithBoc = async (fn: (senderWithBoc: Sender) => Promise<void>): Promise<string> => {
        let boc = "";
        const senderWithBoc: Sender = {
            address: senderAddress,
            send: async (args: SenderArguments) => {
                boc = await sendSignedTransaction(args);
            },
        };
        await fn(senderWithBoc);
        if (!boc) {
            throw new Error("wallet did not return signed boc");
        }
        return boc;
    };

    return {
        address: wallet?.account.address ?? null,
        connected: !!wallet,
        sendWithBoc,
    }
}

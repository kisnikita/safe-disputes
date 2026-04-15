import { useTonConnectUI, useTonWallet } from "@tonconnect/ui-react";
import { SenderArguments, Sender, Address } from "@ton/core";

export function useTonConnect(): {
    sender: Sender;
    address: string | null;
    connected: boolean;
    sendSignedTransaction: (args: SenderArguments) => Promise<string>;
} {
    const [tonConnectUI] = useTonConnectUI()
    const wallet = useTonWallet()

    const sendSignedTransaction = async (args: SenderArguments): Promise<string> => {
        const result = await tonConnectUI.sendTransaction(
            {
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

    return {
        sender: {
            send: async (args: SenderArguments) => {
                await sendSignedTransaction(args);
            },
            address: wallet?.account?.address ? Address.parse(wallet?.account?.address) : undefined
        },
        address: wallet?.account.address ?? null,
        connected: !!wallet,
        sendSignedTransaction,
    }
}

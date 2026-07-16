# Complexity inventory

Generated: 2026-07-16T07:47:27Z
Base: `1e3cc78154a481c4fdd8b35352c4e81824844ee6`
Tools: `gocyclo@v0.6.0`, `gocognit@v1.2.1`, threshold `-over 10`

| Cyclomatic violations | Cognitive violations | Union | Max cycle | Max cognitive |
|---:|---:|---:|---:|---:|
| 53 | 88 | 90 | 48 | 99 |

Generated files are excluded only when their first 40 lines contain the canonical `// Code generated ... DO NOT EDIT.` marker.

| Rank | File | Line | Function | Package | Cyclomatic | Cognitive |
|---:|---|---:|---|---|---:|---:|
| 1 | `relayer/processor/path_processor_internal.go` | 28 | `(*PathProcessor).getMessagesToSend` | `processor` | 42 | 99 |
| 2 | `relayer/processor/path_processor_internal.go` | 741 | `(*PathProcessor).queuePreInitMessages` | `processor` | 48 | 90 |
| 3 | `relayer/processor/path_end_runtime.go` | 285 | `(*pathEndRuntime).shouldTerminate` | `processor` | 45 | 82 |
| 4 | `relayer/processor/path_end_runtime.go` | 150 | `(*pathEndRuntime).mergeMessageCache` | `processor` | 26 | 70 |
| 5 | `cmd/keys.go` | 147 | `keysRestoreCmd` | `cmd` | 23 | 67 |
| 6 | `relayer/processor/path_processor_internal.go` | 1200 | `(*PathProcessor).queuePendingRecvAndAcks` | `processor` | 34 | 63 |
| 7 | `relayer/chains/penumbra/tx.go` | 1795 | `(*PenumbraProvider).acknowledgementsFromResultTx` | `penumbra` | 26 | 48 |
| 8 | `relayer/processor/path_end_runtime.go` | 722 | `(*pathEndRuntime).shouldSendChannelMessage` | `processor` | 24 | 48 |
| 9 | `relayer/naive-strategy.go` | 16 | `UnrelayedSequences` | `relayer` | 23 | 46 |
| 10 | `relayer/chains/cosmos/cosmos_chain_processor.go` | 338 | `(*CosmosChainProcessor).queryCycle` | `cosmos` | 24 | 43 |
| 11 | `relayer/processor/path_processor_internal.go` | 1550 | `(*PathProcessor).shouldTerminateForFlushComplete` | `processor` | 24 | 43 |
| 12 | `relayer/processor/path_processor_internal.go` | 1422 | `(*PathProcessor).flush` | `processor` | 20 | 40 |
| 13 | `relayer/chains/cosmos/feegrant.go` | 249 | `(*CosmosProvider).EnsureBasicGrants` | `cosmos` | 25 | 39 |
| 14 | `cmd/tx.go` | 585 | `xfersend` | `cmd` | 19 | 36 |
| 15 | `relayer/client.go` | 118 | `CreateClient` | `relayer` | 21 | 32 |
| 16 | `relayer/chains/cosmos/tx.go` | 1395 | `(*CosmosProvider).RelayPacketFromSequence` | `cosmos` | 14 | 29 |
| 17 | `relayer/chains/penumbra/tx.go` | 1682 | `(*PenumbraProvider).RelayPacketFromSequence` | `penumbra` | 14 | 29 |
| 18 | `relayer/chains/cosmos/provider.go` | 330 | `(*CosmosProvider).startLivelinessChecks` | `cosmos` | 11 | 28 |
| 19 | `relayer/chains/penumbra/provider.go` | 283 | `(*PenumbraProvider).startLivelinessChecks` | `penumbra` | 11 | 28 |
| 20 | `relayer/processor/path_end_runtime.go` | 868 | `(*pathEndRuntime).trackProcessingMessage` | `processor` | 16 | 27 |
| 21 | `cmd/query.go` | 329 | `queryBalancesCmd` | `cmd` | 12 | 27 |
| 22 | `relayer/chains/penumbra/penumbra_chain_processor.go` | 284 | `(*PenumbraChainProcessor).queryCycle` | `penumbra` | 14 | 25 |
| 23 | `relayer/processor/message_processor.go` | 415 | `(*messageProcessor).sendBatchMessages` | `processor` | 14 | 25 |
| 24 | `cmd/keys.go` | 77 | `keysAddCmd` | `cmd` | 12 | 25 |
| 25 | `relayer/processor/path_processor.go` | 423 | `(*PathProcessor).Run` | `processor` | 17 | 24 |
| 26 | `interchaintest/stride/setup_test.go` | 120 | `ModifyGenesisStride` | `stride_test` | 13 | 24 |
| 27 | `relayer/processor/path_end_runtime.go` | 953 | `(*pathEndRuntime).trackFinishedProcessingMessage` | `processor` | 16 | 23 |
| 28 | `relayer/naive-strategy.go` | 244 | `UnrelayedAcknowledgements` | `relayer` | 14 | 23 |
| 29 | `relayer/processor/path_processor_internal.go` | 189 | `(*PathProcessor).unrelayedPacketFlowMessages` | `processor` | 14 | 22 |
| 30 | `cmd/query.go` | 957 | `queryChannelsPaginated` | `cmd` | 13 | 22 |
| 31 | `relayer/strategies.go` | 259 | `relayerStartLegacy` | `relayer` | 12 | 21 |
| 32 | `cmd/query.go` | 906 | `queryChannelsToChain` | `cmd` | 10 | 21 |
| 33 | `relayer/chains/cosmos/tx.go` | 615 | `(*CosmosProvider).buildMessages` | `cosmos` | 15 | 20 |
| 34 | `cmd/query.go` | 1225 | `queryClientsExpiration` | `cmd` | 12 | 20 |
| 35 | `cmd/tx.go` | 197 | `createChannelCmd` | `cmd` | 11 | 20 |
| 36 | `cmd/tx.go` | 425 | `flushCmd` | `cmd` | 11 | 20 |
| 37 | `relayer/chains/cosmos/tx.go` | 565 | `(*CosmosProvider).buildSignerConfig` | `cosmos` | 10 | 20 |
| 38 | `relayer/naive-strategy.go` | 385 | `RelayAcknowledgements` | `relayer` | 14 | 18 |
| 39 | `relayer/chains/cosmos/tx.go` | 1640 | `(*CosmosProvider).PrepareFactory` | `cosmos` | 13 | 18 |
| 40 | `relayer/chains/cosmos/feegrant.go` | 65 | `(*CosmosProvider).GetGranteeValidBasicGrants` | `cosmos` | 10 | 18 |
| 41 | `relayer/chains/cosmos/cosmos_chain_processor.go` | 211 | `(*CosmosChainProcessor).Run` | `cosmos` | 14 | 17 |
| 42 | `relayer/chains/cosmos/tx.go` | 428 | `(*CosmosProvider).waitForTx` | `cosmos` | 10 | 17 |
| 43 | `relayer/chains/penumbra/tx.go` | 103 | `msgToPenumbraAction` | `penumbra` | 17 | 2 |
| 44 | `relayer/chains/cosmos/tx.go` | 266 | `(*CosmosProvider).SendMsgsWith` | `cosmos` | 15 | 16 |
| 45 | `relayer/chains/cosmos/message_handlers.go` | 66 | `(*CosmosChainProcessor).handleChannelMessage` | `cosmos` | 11 | 16 |
| 46 | `relayer/chains/penumbra/message_handlers.go` | 51 | `(*PenumbraChainProcessor).handleChannelMessage` | `penumbra` | 11 | 16 |
| 47 | `relayer/query.go` | 234 | `QueryBalance` | `relayer` | 11 | 16 |
| 48 | `cmd/query.go` | 258 | `queryBalanceCmd` | `cmd` | 10 | 16 |
| 49 | `cmd/query.go` | 400 | `queryHeaderCmd` | `cmd` | 10 | 16 |
| 50 | `relayer/chains/cosmos/query.go` | 48 | `(*CosmosProvider).queryIBCMessages` | `cosmos` | 10 | 16 |
| 51 | `relayer/relayMsgs.go` | 178 | `(*RelayMsgs).send` | `relayer` | 10 | 16 |
| 52 | `cmd/tx.go` | 711 | `setPathsFromArgs` | `cmd` | 15 | 14 |
| 53 | `relayer/chains/penumbra/penumbra_chain_processor.go` | 159 | `(*PenumbraChainProcessor).Run` | `penumbra` | 12 | 15 |
| 54 | `relayer/chains/cosmos/query.go` | 237 | `(*CosmosProvider).QueryFeegrantsByGranter` | `cosmos` | 9 | 15 |
| 55 | `cmd/query.go` | 800 | `queryChannel` | `cmd` | 8 | 15 |
| 56 | `cmd/query.go` | 504 | `queryClientCmd` | `cmd` | 8 | 15 |
| 57 | `cmd/query.go` | 650 | `queryConnectionsUsingClient` | `cmd` | 8 | 15 |
| 58 | `relayer/chains/cosmos/log.go` | 21 | `getChannelsIfPresent` | `cosmos` | 6 | 15 |
| 59 | `relayer/chains/penumbra/log.go` | 18 | `getChannelsIfPresent` | `penumbra` | 6 | 15 |
| 60 | `relayer/processor/message_processor.go` | 327 | `(*messageProcessor).trackAndSendMessages` | `processor` | 12 | 14 |
| 61 | `relayer/chains/cosmos/grpc_query.go` | 34 | `(*CosmosProvider).Invoke` | `cosmos` | 11 | 14 |
| 62 | `relayer/chains/penumbra/grpc_query.go` | 34 | `(*PenumbraProvider).Invoke` | `penumbra` | 11 | 14 |
| 63 | `relayer/path.go` | 328 | `(*Path).QueryPathStatus` | `relayer` | 14 | 11 |
| 64 | `relayer/processor/message_processor.go` | 513 | `(*messageProcessor).sendSingleMessage` | `processor` | 9 | 14 |
| 65 | `cmd/tx.go` | 289 | `closeChannelCmd` | `cmd` | 8 | 14 |
| 66 | `relayer/chains/cosmos/query.go` | 198 | `(*CosmosProvider).QueryFeegrantsByGrantee` | `cosmos` | 8 | 14 |
| 67 | `relayer/processor/path_end_runtime.go` | 528 | `(*pathEndRuntime).shouldSendPacketMessage` | `processor` | 13 | 13 |
| 68 | `relayer/processor/path_end.go` | 38 | `(PathEnd).checkChannelMatch` | `processor` | 10 | 13 |
| 69 | `relayer/processor/path_end_runtime.go` | 650 | `(*pathEndRuntime).shouldSendConnectionMessage` | `processor` | 13 | 10 |
| 70 | `relayer/processor/path_processor_internal.go` | 456 | `(*PathProcessor).unrelayedChannelHandshakeMessages` | `processor` | 10 | 13 |
| 71 | `relayer/processor/path_processor_internal.go` | 327 | `(*PathProcessor).unrelayedConnectionHandshakeMessages` | `processor` | 10 | 13 |
| 72 | `relayer/chains/cosmos/tx.go` | 527 | `parseEventsFromTxResponse` | `cosmos` | 8 | 13 |
| 73 | `relayer/processor/path_end_runtime.go` | 269 | `(*pathEndRuntime).handleCallbacks` | `processor` | 6 | 13 |
| 74 | `relayer/client.go` | 19 | `(*Chain).CreateClients` | `relayer` | 10 | 12 |
| 75 | `relayer/chains/cosmos/tx.go` | 376 | `(*CosmosProvider).broadcastTx` | `cosmos` | 9 | 12 |
| 76 | `relayer/chains/penumbra/keys.go` | 88 | `(*PenumbraProvider).KeyAddOrRestore` | `penumbra` | 9 | 12 |
| 77 | `relayer/relaymsgs_test.go` | 148 | `TestRelayMsgs_Send_Errors` | `relayer_test` | 9 | 12 |
| 78 | `relayer/chains/cosmos/feegrant.go` | 22 | `(*CosmosProvider).GetValidBasicGrants` | `cosmos` | 8 | 12 |
| 79 | `cmd/query.go` | 1173 | `queryUnrelayedAcknowledgements` | `cmd` | 7 | 12 |
| 80 | `cmd/query.go` | 1120 | `queryUnrelayedPackets` | `cmd` | 7 | 12 |
| 81 | `relayer/processor/path_end_runtime.go` | 1052 | `(*pathEndRuntime).ShouldRelayChannel` | `processor` | 7 | 12 |
| 82 | `relayer/chains/cosmos/feegrant.go` | 366 | `(*CosmosProvider).GrantAllGranteesBasicAllowance` | `cosmos` | 11 | 11 |
| 83 | `relayer/chains/cosmos/feegrant.go` | 403 | `(*CosmosProvider).GrantAllGranteesBasicAllowanceWithExpiration` | `cosmos` | 11 | 11 |
| 84 | `relayer/processor/message_processor.go` | 254 | `(*messageProcessor).assembleMsgUpdateClient` | `processor` | 10 | 11 |
| 85 | `relayer/chains/penumbra/tx.go` | 2165 | `(*PenumbraProvider).broadcastTx` | `penumbra` | 8 | 11 |
| 86 | `relayer/chains/cosmos/query.go` | 736 | `(*CosmosProvider).QueryConnectionsUsingClient` | `cosmos` | 7 | 11 |
| 87 | `relayer/strategies.go` | 412 | `applyChannelFilterRule` | `relayer` | 7 | 11 |
| 88 | `cmd/keys.go` | 269 | `keysDeleteCmd` | `cmd` | 6 | 11 |
| 89 | `cmd/query.go` | 750 | `queryConnectionChannels` | `cmd` | 6 | 11 |
| 90 | `relayer/processor/path_processor_internal.go` | 916 | `(*PathProcessor).processLatestMessages` | `processor` | 6 | 11 |

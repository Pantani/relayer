# Complexity inventory

Generated: 2026-07-16T15:50:53Z
Base: `ee32f2940f00ea94352717b518b6ddcfb197268e`
Tools: `gocyclo@v0.6.0`, `gocognit@v1.2.1`, threshold `-over 10`

| Cyclomatic violations | Cognitive violations | Union | Max cycle | Max cognitive |
|---:|---:|---:|---:|---:|
| 42 | 67 | 69 | 48 | 99 |

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
| 14 | `relayer/client.go` | 118 | `CreateClient` | `relayer` | 21 | 32 |
| 15 | `relayer/chains/cosmos/tx.go` | 1395 | `(*CosmosProvider).RelayPacketFromSequence` | `cosmos` | 14 | 29 |
| 16 | `relayer/chains/penumbra/tx.go` | 1682 | `(*PenumbraProvider).RelayPacketFromSequence` | `penumbra` | 14 | 29 |
| 17 | `relayer/processor/path_end_runtime.go` | 868 | `(*pathEndRuntime).trackProcessingMessage` | `processor` | 16 | 27 |
| 18 | `relayer/chains/penumbra/penumbra_chain_processor.go` | 284 | `(*PenumbraChainProcessor).queryCycle` | `penumbra` | 14 | 25 |
| 19 | `relayer/processor/message_processor.go` | 415 | `(*messageProcessor).sendBatchMessages` | `processor` | 14 | 25 |
| 20 | `cmd/keys.go` | 77 | `keysAddCmd` | `cmd` | 12 | 25 |
| 21 | `relayer/processor/path_processor.go` | 423 | `(*PathProcessor).Run` | `processor` | 17 | 24 |
| 22 | `interchaintest/stride/setup_test.go` | 120 | `ModifyGenesisStride` | `stride_test` | 13 | 24 |
| 23 | `relayer/processor/path_end_runtime.go` | 953 | `(*pathEndRuntime).trackFinishedProcessingMessage` | `processor` | 16 | 23 |
| 24 | `relayer/naive-strategy.go` | 244 | `UnrelayedAcknowledgements` | `relayer` | 14 | 23 |
| 25 | `relayer/processor/path_processor_internal.go` | 189 | `(*PathProcessor).unrelayedPacketFlowMessages` | `processor` | 14 | 22 |
| 26 | `relayer/strategies.go` | 259 | `relayerStartLegacy` | `relayer` | 12 | 21 |
| 27 | `relayer/chains/cosmos/tx.go` | 615 | `(*CosmosProvider).buildMessages` | `cosmos` | 15 | 20 |
| 28 | `relayer/chains/cosmos/tx.go` | 565 | `(*CosmosProvider).buildSignerConfig` | `cosmos` | 10 | 20 |
| 29 | `relayer/naive-strategy.go` | 385 | `RelayAcknowledgements` | `relayer` | 14 | 18 |
| 30 | `relayer/chains/cosmos/tx.go` | 1640 | `(*CosmosProvider).PrepareFactory` | `cosmos` | 13 | 18 |
| 31 | `relayer/chains/cosmos/feegrant.go` | 65 | `(*CosmosProvider).GetGranteeValidBasicGrants` | `cosmos` | 10 | 18 |
| 32 | `relayer/chains/cosmos/cosmos_chain_processor.go` | 211 | `(*CosmosChainProcessor).Run` | `cosmos` | 14 | 17 |
| 33 | `relayer/chains/cosmos/tx.go` | 428 | `(*CosmosProvider).waitForTx` | `cosmos` | 10 | 17 |
| 34 | `relayer/chains/penumbra/tx.go` | 103 | `msgToPenumbraAction` | `penumbra` | 17 | 2 |
| 35 | `relayer/chains/cosmos/tx.go` | 266 | `(*CosmosProvider).SendMsgsWith` | `cosmos` | 15 | 16 |
| 36 | `relayer/query.go` | 234 | `QueryBalance` | `relayer` | 11 | 16 |
| 37 | `relayer/chains/cosmos/query.go` | 48 | `(*CosmosProvider).queryIBCMessages` | `cosmos` | 10 | 16 |
| 38 | `relayer/relayMsgs.go` | 178 | `(*RelayMsgs).send` | `relayer` | 10 | 16 |
| 39 | `relayer/chains/penumbra/penumbra_chain_processor.go` | 159 | `(*PenumbraChainProcessor).Run` | `penumbra` | 12 | 15 |
| 40 | `relayer/chains/cosmos/query.go` | 237 | `(*CosmosProvider).QueryFeegrantsByGranter` | `cosmos` | 9 | 15 |
| 41 | `relayer/chains/cosmos/log.go` | 21 | `getChannelsIfPresent` | `cosmos` | 6 | 15 |
| 42 | `relayer/chains/penumbra/log.go` | 18 | `getChannelsIfPresent` | `penumbra` | 6 | 15 |
| 43 | `relayer/processor/message_processor.go` | 327 | `(*messageProcessor).trackAndSendMessages` | `processor` | 12 | 14 |
| 44 | `relayer/chains/cosmos/grpc_query.go` | 34 | `(*CosmosProvider).Invoke` | `cosmos` | 11 | 14 |
| 45 | `relayer/chains/penumbra/grpc_query.go` | 34 | `(*PenumbraProvider).Invoke` | `penumbra` | 11 | 14 |
| 46 | `relayer/path.go` | 328 | `(*Path).QueryPathStatus` | `relayer` | 14 | 11 |
| 47 | `relayer/processor/message_processor.go` | 513 | `(*messageProcessor).sendSingleMessage` | `processor` | 9 | 14 |
| 48 | `relayer/chains/cosmos/query.go` | 198 | `(*CosmosProvider).QueryFeegrantsByGrantee` | `cosmos` | 8 | 14 |
| 49 | `relayer/processor/path_end_runtime.go` | 528 | `(*pathEndRuntime).shouldSendPacketMessage` | `processor` | 13 | 13 |
| 50 | `relayer/processor/path_end.go` | 38 | `(PathEnd).checkChannelMatch` | `processor` | 10 | 13 |
| 51 | `relayer/processor/path_end_runtime.go` | 650 | `(*pathEndRuntime).shouldSendConnectionMessage` | `processor` | 13 | 10 |
| 52 | `relayer/processor/path_processor_internal.go` | 456 | `(*PathProcessor).unrelayedChannelHandshakeMessages` | `processor` | 10 | 13 |
| 53 | `relayer/processor/path_processor_internal.go` | 327 | `(*PathProcessor).unrelayedConnectionHandshakeMessages` | `processor` | 10 | 13 |
| 54 | `relayer/chains/cosmos/tx.go` | 527 | `parseEventsFromTxResponse` | `cosmos` | 8 | 13 |
| 55 | `relayer/processor/path_end_runtime.go` | 269 | `(*pathEndRuntime).handleCallbacks` | `processor` | 6 | 13 |
| 56 | `relayer/client.go` | 19 | `(*Chain).CreateClients` | `relayer` | 10 | 12 |
| 57 | `relayer/chains/cosmos/tx.go` | 376 | `(*CosmosProvider).broadcastTx` | `cosmos` | 9 | 12 |
| 58 | `relayer/chains/penumbra/keys.go` | 88 | `(*PenumbraProvider).KeyAddOrRestore` | `penumbra` | 9 | 12 |
| 59 | `relayer/relaymsgs_test.go` | 148 | `TestRelayMsgs_Send_Errors` | `relayer_test` | 9 | 12 |
| 60 | `relayer/chains/cosmos/feegrant.go` | 22 | `(*CosmosProvider).GetValidBasicGrants` | `cosmos` | 8 | 12 |
| 61 | `relayer/processor/path_end_runtime.go` | 1052 | `(*pathEndRuntime).ShouldRelayChannel` | `processor` | 7 | 12 |
| 62 | `relayer/chains/cosmos/feegrant.go` | 366 | `(*CosmosProvider).GrantAllGranteesBasicAllowance` | `cosmos` | 11 | 11 |
| 63 | `relayer/chains/cosmos/feegrant.go` | 403 | `(*CosmosProvider).GrantAllGranteesBasicAllowanceWithExpiration` | `cosmos` | 11 | 11 |
| 64 | `relayer/processor/message_processor.go` | 254 | `(*messageProcessor).assembleMsgUpdateClient` | `processor` | 10 | 11 |
| 65 | `relayer/chains/penumbra/tx.go` | 2165 | `(*PenumbraProvider).broadcastTx` | `penumbra` | 8 | 11 |
| 66 | `relayer/chains/cosmos/query.go` | 736 | `(*CosmosProvider).QueryConnectionsUsingClient` | `cosmos` | 7 | 11 |
| 67 | `relayer/strategies.go` | 412 | `applyChannelFilterRule` | `relayer` | 7 | 11 |
| 68 | `cmd/keys.go` | 269 | `keysDeleteCmd` | `cmd` | 6 | 11 |
| 69 | `relayer/processor/path_processor_internal.go` | 916 | `(*PathProcessor).processLatestMessages` | `processor` | 6 | 11 |

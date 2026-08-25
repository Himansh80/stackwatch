/**
 * Tier 13 Phase 4 — ServersScreen.
 * FlatList of servers with status badges + pull-to-refresh.
 */
import React, { useCallback, useEffect, useState } from 'react';
import { View, Text, FlatList, RefreshControl, TouchableOpacity, StyleSheet } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { colors, spacing, radius, typography } from '../lib/theme';
import { apiFetch } from '../lib/api';

interface Server {
  id: string;
  name: string;
  hostname: string;
  ip_address: string;
  status: string; // 'up' | 'down' | 'stale' | 'unknown'
  os: string;
  last_seen_at: string;
}

export default function ServersScreen() {
  const [servers, setServers] = useState<Server[]>([]);
  const [refreshing, setRefreshing] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    setError(null);
    try {
      const data = await apiFetch<{ servers: Server[] }>('/api/v1/servers');
      setServers(data.servers ?? []);
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load');
    }
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  const onRefresh = useCallback(async () => {
    setRefreshing(true);
    await load();
    setRefreshing(false);
  }, [load]);

  const renderItem = ({ item }: { item: Server }) => <ServerRow server={item} />;

  return (
    <SafeAreaView style={styles.root}>
      <View style={styles.header}>
        <Text style={typography.h2}>Servers</Text>
        <Text style={typography.caption}>{servers.length} total</Text>
      </View>
      {error && <Text style={styles.error}>{error}</Text>}
      <FlatList
        data={servers}
        keyExtractor={(s) => s.id}
        renderItem={renderItem}
        contentContainerStyle={styles.list}
        refreshControl={<RefreshControl refreshing={refreshing} onRefresh={onRefresh} tintColor={colors.text} />}
        ListEmptyComponent={
          <View style={styles.empty}>
            <Text style={typography.h3}>No servers yet</Text>
            <Text style={typography.caption}>Install an agent on your server to get started</Text>
          </View>
        }
      />
    </SafeAreaView>
  );
}

function ServerRow({ server }: { server: Server }) {
  const statusColor =
    server.status === 'up' ? colors.up
    : server.status === 'down' ? colors.down
    : server.status === 'stale' ? colors.stale
    : colors.unknown;

  return (
    <TouchableOpacity style={styles.row}>
      <View style={[styles.statusDot, { backgroundColor: statusColor }]} />
      <View style={styles.rowBody}>
        <Text style={styles.rowName}>{server.name}</Text>
        <Text style={typography.caption}>{server.hostname} · {server.os}</Text>
      </View>
      <Text style={[styles.statusBadge, { color: statusColor }]}>{server.status.toUpperCase()}</Text>
    </TouchableOpacity>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: colors.bg },
  header: { padding: spacing.lg, paddingBottom: spacing.md },
  list: { padding: spacing.md },
  row: { flexDirection: 'row', alignItems: 'center', backgroundColor: colors.surface, padding: spacing.md, borderRadius: radius.md, marginBottom: spacing.sm },
  statusDot: { width: 10, height: 10, borderRadius: 5, marginRight: spacing.md },
  rowBody: { flex: 1 },
  rowName: { ...typography.h3, color: colors.text },
  statusBadge: { ...typography.caption, fontWeight: '700' },
  empty: { alignItems: 'center', padding: spacing.xxl },
  error: { ...typography.caption, color: colors.danger, padding: spacing.lg },
});

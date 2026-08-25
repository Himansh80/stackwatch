/**
 * Tier 13 Phase 4 — AlertsScreen.
 * Firing alerts with severity badges + acknowledge/resolve buttons.
 */
import React, { useCallback, useEffect, useState } from 'react';
import { View, Text, FlatList, RefreshControl, TouchableOpacity, StyleSheet, Alert } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { colors, spacing, radius, typography } from '../lib/theme';
import { apiFetch } from '../lib/api';

interface AlertItem {
  id: string;
  severity: 'info' | 'warning' | 'critical';
  state: 'open' | 'acknowledged' | 'resolved';
  rule_name: string;
  message: string;
  server_name?: string;
  fired_at: string;
}

export default function AlertsScreen() {
  const [alerts, setAlerts] = useState<AlertItem[]>([]);
  const [refreshing, setRefreshing] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    setError(null);
    try {
      const data = await apiFetch<{ alerts: AlertItem[] }>('/api/v1/alerts?state=open');
      setAlerts(data.alerts ?? []);
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

  const ack = async (id: string) => {
    try {
      await apiFetch(`/api/v1/alerts/${id}/acknowledge`, { method: 'POST' });
      await load();
    } catch (e) {
      Alert.alert('Error', e instanceof Error ? e.message : 'Failed');
    }
  };

  const resolve = async (id: string) => {
    try {
      await apiFetch(`/api/v1/alerts/${id}/resolve`, { method: 'POST' });
      await load();
    } catch (e) {
      Alert.alert('Error', e instanceof Error ? e.message : 'Failed');
    }
  };

  return (
    <SafeAreaView style={styles.root}>
      <View style={styles.header}>
        <Text style={typography.h2}>Alerts</Text>
        <Text style={typography.caption}>{alerts.length} firing</Text>
      </View>
      {error && <Text style={styles.error}>{error}</Text>}
      <FlatList
        data={alerts}
        keyExtractor={(a) => a.id}
        contentContainerStyle={styles.list}
        renderItem={({ item }) => <AlertRow alert={item} onAck={ack} onResolve={resolve} />}
        refreshControl={<RefreshControl refreshing={refreshing} onRefresh={onRefresh} tintColor={colors.text} />}
        ListEmptyComponent={
          <View style={styles.empty}>
            <Text style={typography.h3}>All clear</Text>
            <Text style={typography.caption}>No firing alerts</Text>
          </View>
        }
      />
    </SafeAreaView>
  );
}

function AlertRow({ alert, onAck, onResolve }: { alert: AlertItem; onAck: (id: string) => void; onResolve: (id: string) => void }) {
  const sevColor =
    alert.severity === 'critical' ? colors.danger
    : alert.severity === 'warning' ? colors.warning
    : colors.info;
  return (
    <View style={[styles.row, { borderLeftColor: sevColor }]}>
      <View style={styles.rowHead}>
        <Text style={[styles.severity, { color: sevColor }]}>{alert.severity.toUpperCase()}</Text>
        <Text style={typography.caption}>{alert.fired_at}</Text>
      </View>
      <Text style={typography.h3}>{alert.rule_name}</Text>
      <Text style={[typography.body, { color: colors.textMuted, marginTop: spacing.xs }]}>{alert.message}</Text>
      {alert.server_name && <Text style={typography.caption}>{alert.server_name}</Text>}
      <View style={styles.actions}>
        <TouchableOpacity style={[styles.btn, styles.btnAck]} onPress={() => onAck(alert.id)}>
          <Text style={styles.btnText}>Ack</Text>
        </TouchableOpacity>
        <TouchableOpacity style={[styles.btn, styles.btnResolve]} onPress={() => onResolve(alert.id)}>
          <Text style={styles.btnText}>Resolve</Text>
        </TouchableOpacity>
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: colors.bg },
  header: { padding: spacing.lg, paddingBottom: spacing.md },
  list: { padding: spacing.md },
  row: { backgroundColor: colors.surface, borderRadius: radius.md, padding: spacing.md, marginBottom: spacing.sm, borderLeftWidth: 4 },
  rowHead: { flexDirection: 'row', justifyContent: 'space-between', marginBottom: spacing.xs },
  severity: { ...typography.caption, fontWeight: '700' },
  actions: { flexDirection: 'row', gap: spacing.sm, marginTop: spacing.md },
  btn: { flex: 1, padding: spacing.sm, borderRadius: radius.sm, alignItems: 'center' },
  btnAck: { backgroundColor: colors.surfaceAlt },
  btnResolve: { backgroundColor: colors.primary },
  btnText: { ...typography.caption, color: colors.text, fontWeight: '600' },
  empty: { alignItems: 'center', padding: spacing.xxl },
  error: { ...typography.caption, color: colors.danger, padding: spacing.lg },
});

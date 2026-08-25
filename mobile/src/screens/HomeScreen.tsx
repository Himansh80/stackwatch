/**
 * Tier 13 Phase 4 — HomeScreen.
 * KPI strip + quick actions + recent alerts. Pull-to-refresh.
 */
import React, { useCallback, useEffect, useState } from 'react';
import { View, Text, ScrollView, RefreshControl, TouchableOpacity, StyleSheet, ActivityIndicator } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useNavigation } from '@react-navigation/native';
import type { NativeStackNavigationProp } from '@react-navigation/native-stack';
import { colors, spacing, radius, typography } from '../lib/theme';
import { useAuth } from '../context/AuthContext';
import { apiFetch } from '../lib/api';

interface FleetSummary {
  total_servers: number;
  servers_up: number;
  servers_down: number;
  open_alerts: number;
}

export default function HomeScreen() {
  const nav = useNavigation<NativeStackNavigationProp<any>>();
  const { user } = useAuth();
  const [summary, setSummary] = useState<FleetSummary | null>(null);
  const [refreshing, setRefreshing] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    setError(null);
    try {
      const data = await apiFetch<FleetSummary>('/api/v1/fleet/summary');
      setSummary(data);
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

  return (
    <SafeAreaView style={styles.root}>
      <ScrollView
        contentContainerStyle={styles.scroll}
        refreshControl={<RefreshControl refreshing={refreshing} onRefresh={onRefresh} tintColor={colors.text} />}
      >
        <View style={styles.header}>
          <Text style={typography.h2}>Welcome back</Text>
          <Text style={typography.caption}>{user?.full_name ?? user?.email ?? 'Operator'}</Text>
        </View>

        {error && (
          <View style={styles.errorBanner}>
            <Text style={styles.errorText}>{error}</Text>
          </View>
        )}

        <View style={styles.kpis}>
          <KpiCard label="Servers up" value={summary?.servers_up ?? 0} accent="up" />
          <KpiCard label="Servers down" value={summary?.servers_down ?? 0} accent="down" />
          <KpiCard label="Open alerts" value={summary?.open_alerts ?? 0} accent="warning" />
          <KpiCard label="Total servers" value={summary?.total_servers ?? 0} accent="info" />
        </View>

        <Text style={styles.sectionTitle}>Quick actions</Text>
        <View style={styles.actions}>
          <ActionButton label="View servers" onPress={() => nav.navigate('Servers' as never)} />
          <ActionButton label="View alerts" onPress={() => nav.navigate('Alerts' as never)} />
          <ActionButton label="Open settings" onPress={() => nav.navigate('Settings' as never)} />
        </View>

        <Text style={styles.footnote}>
          Pull down to refresh · Last updated just now
        </Text>
      </ScrollView>
    </SafeAreaView>
  );
}

function KpiCard({ label, value, accent }: { label: string; value: number; accent: 'up' | 'down' | 'warning' | 'info' }) {
  const accentColor =
    accent === 'up' ? colors.success : accent === 'down' ? colors.danger : accent === 'warning' ? colors.warning : colors.info;
  return (
    <View style={[styles.kpi, { borderTopColor: accentColor }]}>
      <Text style={typography.caption}>{label}</Text>
      <Text style={styles.kpiValue}>{value}</Text>
    </View>
  );
}

function ActionButton({ label, onPress }: { label: string; onPress: () => void }) {
  return (
    <TouchableOpacity style={styles.actionBtn} onPress={onPress}>
      <Text style={styles.actionText}>{label}</Text>
    </TouchableOpacity>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: colors.bg },
  scroll: { padding: spacing.lg, paddingBottom: spacing.xxl },
  header: { marginBottom: spacing.lg },
  kpis: { flexDirection: 'row', flexWrap: 'wrap', gap: spacing.md, marginBottom: spacing.xl },
  kpi: { flexBasis: '47%', backgroundColor: colors.surface, borderRadius: radius.md, padding: spacing.md, borderTopWidth: 3 },
  kpiValue: { ...typography.metric, color: colors.text, marginTop: spacing.xs },
  sectionTitle: { ...typography.h3, marginBottom: spacing.md },
  actions: { gap: spacing.sm },
  actionBtn: { backgroundColor: colors.surface, borderRadius: radius.md, padding: spacing.lg, borderLeftWidth: 3, borderLeftColor: colors.primary },
  actionText: { ...typography.h3, color: colors.text },
  footnote: { ...typography.caption, textAlign: 'center', marginTop: spacing.xl },
  errorBanner: { backgroundColor: colors.danger + '22', padding: spacing.md, borderRadius: radius.sm, marginBottom: spacing.md },
  errorText: { ...typography.caption, color: colors.danger },
});

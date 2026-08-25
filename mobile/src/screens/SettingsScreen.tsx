/**
 * Tier 13 Phase 4 — SettingsScreen.
 * Push toggle, backend URL display, app version, send-heartbeat CTA.
 */
import React, { useState } from 'react';
import { View, Text, ScrollView, Switch, TouchableOpacity, StyleSheet, Linking } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import Constants from 'expo-constants';
import { colors, spacing, radius, typography } from '../lib/theme';
import { apiFetch } from '../lib/api';

const BACKEND_URL =
  (Constants.expoConfig?.extra as { backendUrl?: string } | undefined)?.backendUrl ?? 'http://...';

export default function SettingsScreen() {
  const [pushEnabled, setPushEnabled] = useState(true);
  const [sending, setSending] = useState(false);
  const [lastBeat, setLastBeat] = useState<string | null>(null);

  const onBeat = async () => {
    setSending(true);
    try {
      await apiFetch('/api/v1/mobile/devices/heartbeat', {
        method: 'POST',
        body: { device_name: Constants.deviceName ?? 'unknown', app_version: Constants.expoConfig?.version ?? '0.0.0' },
      });
      setLastBeat(new Date().toLocaleTimeString());
    } catch {
      // non-fatal
    } finally {
      setSending(false);
    }
  };

  return (
    <SafeAreaView style={styles.root}>
      <ScrollView contentContainerStyle={styles.scroll}>
        <Text style={typography.h2}>Settings</Text>

        <View style={styles.section}>
          <Text style={typography.h3}>Notifications</Text>
          <View style={styles.row}>
            <Text style={typography.body}>Push notifications</Text>
            <Switch value={pushEnabled} onValueChange={setPushEnabled} />
          </View>
        </View>

        <View style={styles.section}>
          <Text style={typography.h3}>Server</Text>
          <View style={styles.row}>
            <Text style={typography.body}>Backend URL</Text>
            <Text style={typography.caption}>{BACKEND_URL}</Text>
          </View>
          <View style={styles.row}>
            <Text style={typography.body}>App version</Text>
            <Text style={typography.caption}>{Constants.expoConfig?.version ?? '0.0.0'}</Text>
          </View>
          <TouchableOpacity style={styles.btn} onPress={onBeat} disabled={sending}>
            <Text style={styles.btnText}>{sending ? 'Sending…' : 'Send heartbeat now'}</Text>
          </TouchableOpacity>
          {lastBeat && <Text style={typography.caption}>Last beat: {lastBeat}</Text>}
        </View>

        <View style={styles.section}>
          <Text style={typography.h3}>About</Text>
          <Text style={typography.body}>StackWatch Mobile</Text>
          <Text style={typography.caption}>Self-Hosted Monitoring That You Own</Text>
          <TouchableOpacity onPress={() => Linking.openURL('https://stackwatch.smarthomelab.fun')}>
            <Text style={styles.link}>stackwatch.smarthomelab.fun</Text>
          </TouchableOpacity>
        </View>
      </ScrollView>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: colors.bg },
  scroll: { padding: spacing.lg },
  section: { backgroundColor: colors.surface, padding: spacing.lg, borderRadius: radius.lg, marginTop: spacing.lg },
  row: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center', paddingVertical: spacing.sm, borderBottomColor: colors.border, borderBottomWidth: 1 },
  btn: { backgroundColor: colors.primary, padding: spacing.md, borderRadius: radius.md, alignItems: 'center', marginTop: spacing.md },
  btnText: { ...typography.h3, color: colors.text },
  link: { ...typography.body, color: colors.primary, marginTop: spacing.sm },
});

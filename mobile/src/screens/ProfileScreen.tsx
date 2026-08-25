/**
 * Tier 13 Phase 4 — ProfileScreen.
 * User info + tenant + push device list + logout.
 */
import React, { useCallback, useEffect, useState } from 'react';
import { View, Text, ScrollView, TouchableOpacity, StyleSheet, Alert } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { colors, spacing, radius, typography } from '../lib/theme';
import { useAuth } from '../context/AuthContext';
import { mobile, apiFetch } from '../lib/api';

interface PushDevice {
  id: string;
  platform: string;
  device_name: string;
  last_active_at?: string;
  enabled: boolean;
}

export default function ProfileScreen() {
  const { user, logout } = useAuth();
  const [devices, setDevices] = useState<PushDevice[]>([]);

  const load = useCallback(async () => {
    try {
      const d = await mobile.pushDevices();
      setDevices(d.devices as PushDevice[]);
    } catch {
      // non-fatal
    }
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  const onLogout = () =>
    Alert.alert('Sign out?', '', [
      { text: 'Cancel', style: 'cancel' },
      { text: 'Sign out', style: 'destructive', onPress: logout },
    ]);

  const onTestPush = async () => {
    try {
      await mobile.testPush({ title: 'StackWatch test', body: 'Push notifications are working!' });
      Alert.alert('Sent', 'Check your other devices for the test push.');
    } catch (e) {
      Alert.alert('Failed', e instanceof Error ? e.message : 'Could not send');
    }
  };

  return (
    <SafeAreaView style={styles.root}>
      <ScrollView contentContainerStyle={styles.scroll}>
        <View style={styles.avatar}>
          <Text style={styles.avatarText}>
            {(user?.full_name ?? user?.email ?? '?').slice(0, 1).toUpperCase()}
          </Text>
        </View>
        <Text style={styles.name}>{user?.full_name ?? 'Operator'}</Text>
        <Text style={typography.caption}>{user?.email}</Text>

        <View style={styles.card}>
          <Text style={typography.h3}>Push devices</Text>
          {devices.length === 0 ? (
            <Text style={typography.caption}>No devices registered yet</Text>
          ) : (
            devices.map((d) => (
              <View key={d.id} style={styles.deviceRow}>
                <Text style={typography.body}>{d.platform}</Text>
                <Text style={typography.caption}>{d.device_name || 'unnamed'}</Text>
              </View>
            ))
          )}
          <TouchableOpacity style={styles.btn} onPress={onTestPush}>
            <Text style={styles.btnText}>Send test push</Text>
          </TouchableOpacity>
        </View>

        <TouchableOpacity style={[styles.btn, styles.btnDanger]} onPress={onLogout}>
          <Text style={styles.btnText}>Sign out</Text>
        </TouchableOpacity>
      </ScrollView>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: colors.bg },
  scroll: { padding: spacing.lg, alignItems: 'center' },
  avatar: { width: 80, height: 80, borderRadius: 40, backgroundColor: colors.primary, alignItems: 'center', justifyContent: 'center', marginTop: spacing.lg },
  avatarText: { ...typography.h1, color: colors.text, fontSize: 32 },
  name: { ...typography.h2, color: colors.text, marginTop: spacing.md },
  card: { width: '100%', backgroundColor: colors.surface, padding: spacing.lg, borderRadius: radius.lg, marginVertical: spacing.xl },
  deviceRow: { paddingVertical: spacing.sm, borderBottomColor: colors.border, borderBottomWidth: 1 },
  btn: { width: '100%', backgroundColor: colors.primary, padding: spacing.md, borderRadius: radius.md, alignItems: 'center', marginTop: spacing.md },
  btnDanger: { backgroundColor: colors.danger },
  btnText: { ...typography.h3, color: colors.text },
});

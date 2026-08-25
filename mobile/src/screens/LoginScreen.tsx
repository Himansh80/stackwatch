/**
 * Tier 13 Phase 4 — LoginScreen.
 * Email + password + submit. Uses AuthContext.login.
 */
import React, { useState } from 'react';
import { View, Text, TextInput, TouchableOpacity, StyleSheet, ActivityIndicator, KeyboardAvoidingView, Platform } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useAuth } from '../context/AuthContext';
import { colors, spacing, radius, typography } from '../lib/theme';

export default function LoginScreen() {
  const { login, loading } = useAuth();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState<string | null>(null);

  const onSubmit = async () => {
    setError(null);
    try {
      await login(email.trim(), password);
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Login failed');
    }
  };

  return (
    <SafeAreaView style={styles.root}>
      <KeyboardAvoidingView behavior={Platform.OS === 'ios' ? 'padding' : 'height'} style={styles.flex}>
        <View style={styles.brand}>
          <Text style={styles.logo}>StackWatch</Text>
          <Text style={styles.tagline}>Self-Hosted Monitoring That You Own</Text>
        </View>

        <View style={styles.card}>
          <Text style={typography.h3}>Sign in</Text>

          <Text style={styles.label}>Email</Text>
          <TextInput
            style={styles.input}
            placeholder="you@example.com"
            placeholderTextColor={colors.textMuted}
            value={email}
            onChangeText={setEmail}
            autoCapitalize="none"
            keyboardType="email-address"
            autoComplete="email"
          />

          <Text style={styles.label}>Password</Text>
          <TextInput
            style={styles.input}
            placeholder="••••••••"
            placeholderTextColor={colors.textMuted}
            value={password}
            onChangeText={setPassword}
            secureTextEntry
          />

          {error && <Text style={styles.error}>{error}</Text>}

          <TouchableOpacity
            style={[styles.button, loading && styles.buttonDisabled]}
            onPress={onSubmit}
            disabled={loading || !email || !password}
          >
            {loading ? <ActivityIndicator color={colors.text} /> : <Text style={styles.buttonText}>Sign in</Text>}
          </TouchableOpacity>
        </View>

        <Text style={styles.footnote}>Open-source · Self-hosted · Production-ready</Text>
      </KeyboardAvoidingView>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: colors.bg },
  flex: { flex: 1, padding: spacing.lg, justifyContent: 'center' },
  brand: { alignItems: 'center', marginBottom: spacing.xxl },
  logo: { ...typography.h1, color: colors.text, fontSize: 36 },
  tagline: { ...typography.caption, marginTop: spacing.xs },
  card: { backgroundColor: colors.surface, borderRadius: radius.lg, padding: spacing.xl },
  label: { ...typography.caption, color: colors.textMuted, marginTop: spacing.md, marginBottom: spacing.xs },
  input: { backgroundColor: colors.surfaceAlt, borderRadius: radius.md, padding: spacing.md, color: colors.text, fontSize: 16 },
  error: { ...typography.caption, color: colors.danger, marginTop: spacing.md },
  button: { backgroundColor: colors.primary, borderRadius: radius.md, padding: spacing.md, alignItems: 'center', marginTop: spacing.lg },
  buttonDisabled: { opacity: 0.5 },
  buttonText: { ...typography.h3, color: colors.text },
  footnote: { ...typography.caption, textAlign: 'center', marginTop: spacing.xl },
});

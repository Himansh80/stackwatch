/**
 * Tier 13 Phase 5 — Push notification registration hook.
 * Uses expo-notifications to obtain a device token, then POSTs to
 * /api/v1/mobile/push/register. Heartbeats every 5 min while in
 * foreground.
 */
import { useEffect } from 'react';
import * as Notifications from 'expo-notifications';
import Constants from 'expo-constants';
import { Platform } from 'react-native';
import { mobile } from '../lib/api';

// Configure foreground notification display
Notifications.setNotificationHandler({
  handleNotification: async () => ({
    shouldShowAlert: true,
    shouldPlaySound: false,
    shouldSetBadge: true,
    shouldShowBanner: true,
    shouldShowList: true,
  }),
});

export async function registerForPushNotifications(): Promise<string | null> {
  // Android 13+ requires explicit POST_NOTIFICATIONS permission
  if (Platform.OS === 'android') {
    const { status } = await Notifications.getPermissionsAsync();
    if (status !== 'granted') {
      const req = await Notifications.requestPermissionsAsync();
      if (req.status !== 'granted') return null;
    }
  } else {
    const { status } = await Notifications.getPermissionsAsync();
    if (status !== 'granted') {
      const req = await Notifications.requestPermissionsAsync();
      if (req.status !== 'granted') return null;
    }
  }

  const projectId = (Constants.expoConfig?.extra as { eas?: { projectId?: string } } | undefined)?.eas?.projectId;
  const tokenResp = await Notifications.getExpoPushTokenAsync(
    projectId ? { projectId } : undefined,
  );
  return tokenResp.data;
}

export function usePushRegistration(): void {
  useEffect(() => {
    let mounted = true;
    (async () => {
      try {
        const token = await registerForPushNotifications();
        if (!token || !mounted) return;
        const platform = Platform.OS === 'ios' ? 'ios' : Platform.OS === 'android' ? 'android' : 'web';
        await mobile.registerPush({
          platform: platform as 'ios' | 'android' | 'web',
          fcm_token: token,
          app_version: Constants.expoConfig?.version ?? '0.0.0',
          device_name: Constants.deviceName ?? undefined,
        });
      } catch {
        // non-fatal: app still works without push
      }
    })();
    return () => {
      mounted = false;
    };
  }, []);
}

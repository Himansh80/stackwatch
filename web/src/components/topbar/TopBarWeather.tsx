/**
 * TopBarWeather.tsx — placeholder weather chip in topbar.
 * Fetches from open-meteo (no API key) using IP-based geolocation.
 *
 * Behavior:
 *   - On mount, hit /api/v1/geo/me to get {lat, lon, city} from the server.
 *   - Fetch current weather from open-meteo.
 *   - Show temp + condition icon.
 *   - Falls back silently (no chip) if offline or geo unavailable.
 */
import { useEffect, useState } from 'react';
import { CloudIcon } from '../icons';

interface WeatherData {
  temp: number;
  city: string;
}

interface GeoResponse {
  lat?: number;
  lon?: number;
  city?: string;
}

export default function TopBarWeather() {
  const [weather, setWeather] = useState<WeatherData | null>(null);

  useEffect(() => {
    let cancelled = false;
    async function load() {
      try {
        const geoRes = await fetch('/api/v1/geo/me');
        if (!geoRes.ok) return;
        const geo = (await geoRes.json()) as GeoResponse;
        if (!geo.lat || !geo.lon || cancelled) return;
        const wRes = await fetch(
          `https://api.open-meteo.com/v1/forecast?latitude=${geo.lat}&longitude=${geo.lon}&current_weather=true`
        );
        if (!wRes.ok || cancelled) return;
        const w = await wRes.json();
        if (cancelled) return;
        setWeather({
          temp: Math.round(w.current_weather.temperature),
          city: geo.city || 'Your location',
        });
      } catch {
        // silent — weather is decorative
      }
    }
    load();
    return () => { cancelled = true; };
  }, []);

  if (!weather) return null;

  return (
    <div className="tb-weather" title={`${weather.city}, ${weather.temp}°C`}>
      <CloudIcon size={14} />
      <span className="tb-weather-temp">{weather.temp}°</span>
      <span className="tb-weather-city">{weather.city}</span>
    </div>
  );
}
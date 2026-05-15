export const getTimezone = () => {
  return Intl.DateTimeFormat().resolvedOptions().timeZone;
};

export const getTimezoneOffset = () => {
  return new Date().getTimezoneOffset();
};

export const getTimezoneAbbreviation = () => {
  const date = new Date();
  const options = { timeZoneName: 'short' };
  const parts = new Intl.DateTimeFormat('en-US', options).formatToParts(date);
  const tzPart = parts.find(part => part.type === 'timeZoneName');
  return tzPart ? tzPart.value : '';
};

export const convertTimezone = (date, fromZone, toZone) => {
  const sourceDate = new Date(date);
  const sourceOffset = getTimezoneOffsetMinutes(fromZone);
  const targetOffset = getTimezoneOffsetMinutes(toZone);
  const diff = targetOffset - sourceOffset;
  const targetDate = new Date(sourceDate.getTime() + diff * 60 * 1000);
  return targetDate;
};

const getTimezoneOffsetMinutes = (timezone) => {
  const date = new Date();
  const utcDate = new Date(date.toLocaleString('en-US', { timeZone: 'UTC' }));
  const tzDate = new Date(date.toLocaleString('en-US', { timeZone: timezone }));
  return (utcDate - tzDate) / (1000 * 60);
};

export const formatTimezone = () => {
  const timezone = getTimezone();
  const offset = getTimezoneOffset();
  const offsetHours = Math.abs(Math.floor(offset / 60));
  const offsetMinutes = Math.abs(offset % 60);
  const sign = offset <= 0 ? '+' : '-';
  const formatted = `UTC${sign}${offsetHours.toString().padStart(2, '0')}:${offsetMinutes.toString().padStart(2, '0')}`;
  return `${timezone} (${formatted})`;
};

export const getAllTimezones = () => {
  return Intl.supportedValuesOf('timeZone');
};

export const isValidTimezone = (timezone) => {
  try {
    Intl.DateTimeFormat(undefined, { timeZone: timezone });
    return true;
  } catch (e) {
    return false;
  }
};

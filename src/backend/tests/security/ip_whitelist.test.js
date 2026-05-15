const { IPControl } = require('../../../backend/middleware/security/ip_whitelist');

describe('IP Control Tests', () => {
  let ipControl;

  beforeEach(() => {
    ipControl = new IPControl();
  });

  describe('Whitelist Management', () => {
    test('should add IP to whitelist', () => {
      ipControl.addWhitelist('192.168.1.1');
      expect(ipControl.isAllowed('192.168.1.1')).toBe(true);
    });

    test('should add multiple IPs to whitelist', () => {
      ipControl.addWhitelist('192.168.1.1');
      ipControl.addWhitelist('192.168.1.2');
      expect(ipControl.isAllowed('192.168.1.1')).toBe(true);
      expect(ipControl.isAllowed('192.168.1.2')).toBe(true);
    });

    test('should handle CIDR notation', () => {
      ipControl.addWhitelist('192.168.1.0/24');
      expect(ipControl.isAllowed('192.168.1.1')).toBe(true);
      expect(ipControl.isAllowed('192.168.1.100')).toBe(true);
    });
  });

  describe('Blacklist Management', () => {
    test('should add IP to blacklist', () => {
      ipControl.addBlacklist('10.0.0.1');
      expect(ipControl.isAllowed('10.0.0.1')).toBe(false);
    });

    test('should prioritize blacklist over whitelist', () => {
      ipControl.addWhitelist('10.0.0.1');
      ipControl.addBlacklist('10.0.0.1');
      expect(ipControl.isAllowed('10.0.0.1')).toBe(false);
    });

    test('should block IP range in blacklist', () => {
      ipControl.addBlacklist('172.16.0.0/16');
      expect(ipControl.isAllowed('172.16.0.1')).toBe(false);
      expect(ipControl.isAllowed('172.16.255.255')).toBe(false);
      expect(ipControl.isAllowed('172.17.0.1')).toBe(true);
    });
  });

  describe('Default Behavior', () => {
    test('should allow IPs not in any list by default', () => {
      expect(ipControl.isAllowed('8.8.8.8')).toBe(true);
    });

    test('should deny IPs when defaultAction is deny', () => {
      ipControl.setDefaultAction('deny');
      expect(ipControl.isAllowed('8.8.8.8')).toBe(false);
    });
  });

  describe('Edge Cases', () => {
    test('should handle IPv6 addresses', () => {
      ipControl.addWhitelist('::1');
      expect(ipControl.isAllowed('::1')).toBe(true);
    });

    test('should handle invalid IP addresses', () => {
      expect(ipControl.isAllowed('invalid-ip')).toBe(false);
      expect(ipControl.isAllowed('')).toBe(false);
    });

    test('should handle localhost', () => {
      ipControl.addWhitelist('127.0.0.1');
      ipControl.addWhitelist('::1');
      expect(ipControl.isAllowed('127.0.0.1')).toBe(true);
      expect(ipControl.isAllowed('::1')).toBe(true);
    });
  });
});

export namespace config {
	
	export class Settings {
	    username: string;
	    password: string;
	    role: string;
	    isp: string;
	    remember: boolean;
	    autoLogin: boolean;
	    autoStart?: boolean;
	    exitBehavior: string;
	    skipExitPrompt: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Settings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.username = source["username"];
	        this.password = source["password"];
	        this.role = source["role"];
	        this.isp = source["isp"];
	        this.remember = source["remember"];
	        this.autoLogin = source["autoLogin"];
	        this.autoStart = source["autoStart"];
	        this.exitBehavior = source["exitBehavior"];
	        this.skipExitPrompt = source["skipExitPrompt"];
	    }
	}

}

export namespace main {
	
	export class AboutInfo {
	    version: string;
	    sha256: string;
	    project: string;
	    issues: string;
	
	    static createFrom(source: any = {}) {
	        return new AboutInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.version = source["version"];
	        this.sha256 = source["sha256"];
	        this.project = source["project"];
	        this.issues = source["issues"];
	    }
	}
	export class AppState {
	    status: string;
	    message: string;
	    ssid: string;
	    interface: string;
	    ip: string;
	    mac: string;
	    signal: string;
	    provider: string;
	    account: string;
	    lastChecked: string;
	    networks: string[];
	    bytesIn4: number;
	    bytesOut4: number;
	    onlineCount: number;
	    terminals: string[];
	    authCode: string;
	    authMessage: string;
	    dialCode: string;
	    dialMessage: string;
	    downloadRate: number;
	    uploadRate: number;
	
	    static createFrom(source: any = {}) {
	        return new AppState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.message = source["message"];
	        this.ssid = source["ssid"];
	        this.interface = source["interface"];
	        this.ip = source["ip"];
	        this.mac = source["mac"];
	        this.signal = source["signal"];
	        this.provider = source["provider"];
	        this.account = source["account"];
	        this.lastChecked = source["lastChecked"];
	        this.networks = source["networks"];
	        this.bytesIn4 = source["bytesIn4"];
	        this.bytesOut4 = source["bytesOut4"];
	        this.onlineCount = source["onlineCount"];
	        this.terminals = source["terminals"];
	        this.authCode = source["authCode"];
	        this.authMessage = source["authMessage"];
	        this.dialCode = source["dialCode"];
	        this.dialMessage = source["dialMessage"];
	        this.downloadRate = source["downloadRate"];
	        this.uploadRate = source["uploadRate"];
	    }
	}
	export class UpdateInfo {
	    status: string;
	    version: string;
	    name: string;
	    notes: string;
	    url: string;
	    publishedAt: string;
	    assetUrl: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdateInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.version = source["version"];
	        this.name = source["name"];
	        this.notes = source["notes"];
	        this.url = source["url"];
	        this.publishedAt = source["publishedAt"];
	        this.assetUrl = source["assetUrl"];
	    }
	}

}

